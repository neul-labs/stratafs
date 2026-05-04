/*
 * AgentFS Finder Sync Extension
 *
 * Adds context menu items and badges to files in Finder.
 * Build as a Finder Sync Extension target in Xcode.
 */

import Cocoa
import FinderSync

class FinderSync: FIFinderSync {

    var monitoredDirectories: Set<URL> = []

    override init() {
        super.init()

        // Load monitored directories from AgentFS config
        loadMonitoredDirectories()

        // Set monitored directories
        FIFinderSyncController.default().directoryURLs = monitoredDirectories

        // Register for notifications
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(configChanged),
            name: NSNotification.Name("AgentFSConfigChanged"),
            object: nil
        )
    }

    // MARK: - Primary Finder Sync protocol methods

    override func beginObservingDirectory(at url: URL) {
        // Called when Finder begins observing a directory
    }

    override func endObservingDirectory(at url: URL) {
        // Called when Finder stops observing a directory
    }

    override func requestBadgeIdentifier(for url: URL) {
        // Check if file is indexed by AgentFS
        if isFileIndexed(url) {
            FIFinderSyncController.default().setBadgeIdentifier("indexed", for: url)
        }
    }

    // MARK: - Menu and toolbar item support

    override var toolbarItemName: String {
        return "AgentFS"
    }

    override var toolbarItemToolTip: String {
        return "AgentFS Actions"
    }

    override var toolbarItemImage: NSImage {
        return NSImage(named: NSImage.folderSmartName)!
    }

    override func menu(for menuKind: FIMenuKind) -> NSMenu {
        let menu = NSMenu(title: "AgentFS")

        switch menuKind {
        case .contextualMenuForItems:
            // File context menu items
            menu.addItem(withTitle: "View AgentFS Metadata", action: #selector(viewMetadata(_:)), keyEquivalent: "")
            menu.addItem(withTitle: "View Chunks", action: #selector(viewChunks(_:)), keyEquivalent: "")
            menu.addItem(NSMenuItem.separator())
            menu.addItem(withTitle: "Find Similar Files", action: #selector(findSimilar(_:)), keyEquivalent: "")
            menu.addItem(NSMenuItem.separator())
            menu.addItem(withTitle: "Reindex", action: #selector(reindex(_:)), keyEquivalent: "")

        case .contextualMenuForContainer:
            // Folder background menu items
            menu.addItem(withTitle: "Add to AgentFS", action: #selector(addSource(_:)), keyEquivalent: "")
            menu.addItem(withTitle: "Export Metadata Here", action: #selector(exportMetadata(_:)), keyEquivalent: "")

        case .contextualMenuForSidebar:
            // Sidebar items
            break

        case .toolbarItemMenu:
            menu.addItem(withTitle: "Open AgentFS", action: #selector(openApp(_:)), keyEquivalent: "")
            menu.addItem(withTitle: "Search...", action: #selector(search(_:)), keyEquivalent: "")

        @unknown default:
            break
        }

        return menu
    }

    // MARK: - Actions

    @IBAction func viewMetadata(_ sender: AnyObject?) {
        guard let items = FIFinderSyncController.default().selectedItemURLs(), let url = items.first else { return }
        runAgentFS(["file", "info", url.path])
    }

    @IBAction func viewChunks(_ sender: AnyObject?) {
        guard let items = FIFinderSyncController.default().selectedItemURLs(), let url = items.first else { return }
        runAgentFS(["file", "chunks", url.path])
    }

    @IBAction func findSimilar(_ sender: AnyObject?) {
        guard let items = FIFinderSyncController.default().selectedItemURLs(), let url = items.first else { return }
        // Open in browser with similarity search
        let task = Process()
        task.launchPath = "/usr/bin/open"
        task.arguments = ["agentfs://search?similar=\(url.path)"]
        try? task.run()
    }

    @IBAction func reindex(_ sender: AnyObject?) {
        guard let items = FIFinderSyncController.default().selectedItemURLs() else { return }
        for url in items {
            runAgentFS(["file", "reindex", url.path])
        }
        showNotification("Queued \(items.count) file(s) for reindexing")
    }

    @IBAction func addSource(_ sender: AnyObject?) {
        guard let url = FIFinderSyncController.default().targetedURL() else { return }
        runAgentFS(["source", "add", "--path", url.path])
        showNotification("Added source: \(url.lastPathComponent)")
    }

    @IBAction func exportMetadata(_ sender: AnyObject?) {
        guard let url = FIFinderSyncController.default().targetedURL() else { return }
        runAgentFS(["fs", "export", "--output", url.path])
        showNotification("Exported metadata to: \(url.lastPathComponent)")
    }

    @IBAction func openApp(_ sender: AnyObject?) {
        NSWorkspace.shared.launchApplication("AgentFS")
    }

    @IBAction func search(_ sender: AnyObject?) {
        let task = Process()
        task.launchPath = "/usr/bin/open"
        task.arguments = ["agentfs://search"]
        try? task.run()
    }

    // MARK: - Helper methods

    private func loadMonitoredDirectories() {
        let homeDir = FileManager.default.homeDirectoryForCurrentUser
        let configPath = homeDir.appendingPathComponent(".agentfs/config.json")

        guard let data = try? Data(contentsOf: configPath),
              let config = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let sources = config["sources"] as? [[String: Any]] else {
            return
        }

        for source in sources {
            if let path = source["path"] as? String,
               let enabled = source["enabled"] as? Bool,
               enabled {
                monitoredDirectories.insert(URL(fileURLWithPath: path))
            }
        }
    }

    private func isFileIndexed(_ url: URL) -> Bool {
        let homeDir = FileManager.default.homeDirectoryForCurrentUser
        let dbPath = homeDir.appendingPathComponent(".agentfs/agentfs.db").path

        // Quick check via SQLite
        var db: OpaquePointer?
        guard sqlite3_open(dbPath, &db) == SQLITE_OK else { return false }
        defer { sqlite3_close(db) }

        let query = "SELECT 1 FROM files WHERE path = ? AND deleted_at IS NULL LIMIT 1"
        var stmt: OpaquePointer?
        guard sqlite3_prepare_v2(db, query, -1, &stmt, nil) == SQLITE_OK else { return false }
        defer { sqlite3_finalize(stmt) }

        sqlite3_bind_text(stmt, 1, url.path, -1, nil)
        return sqlite3_step(stmt) == SQLITE_ROW
    }

    private func runAgentFS(_ arguments: [String]) {
        let task = Process()
        task.launchPath = "/usr/local/bin/agentfs"
        task.arguments = arguments

        let pipe = Pipe()
        task.standardOutput = pipe
        task.standardError = pipe

        try? task.run()
        task.waitUntilExit()

        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        if let output = String(data: data, encoding: .utf8), !output.isEmpty {
            showAlert(output)
        }
    }

    private func showNotification(_ message: String) {
        let notification = NSUserNotification()
        notification.title = "AgentFS"
        notification.informativeText = message
        NSUserNotificationCenter.default.deliver(notification)
    }

    private func showAlert(_ message: String) {
        DispatchQueue.main.async {
            let alert = NSAlert()
            alert.messageText = "AgentFS"
            alert.informativeText = message
            alert.alertStyle = .informational
            alert.runModal()
        }
    }

    @objc private func configChanged() {
        loadMonitoredDirectories()
        FIFinderSyncController.default().directoryURLs = monitoredDirectories
    }
}
