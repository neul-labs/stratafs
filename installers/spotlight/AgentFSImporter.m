/*
 * AgentFS Spotlight Importer
 *
 * This Spotlight importer plugin indexes AgentFS metadata into macOS Spotlight.
 * It reads the AgentFS database and exposes semantic metadata for search.
 *
 * Build: xcodebuild -project AgentFSImporter.xcodeproj
 * Install to: ~/Library/Spotlight/ or /Library/Spotlight/
 */

#import <CoreFoundation/CoreFoundation.h>
#import <CoreServices/CoreServices.h>
#import <Foundation/Foundation.h>
#import <sqlite3.h>

// Plugin entry point
Boolean GetMetadataForFile(void *thisInterface,
                           CFMutableDictionaryRef attributes,
                           CFStringRef contentTypeUTI,
                           CFStringRef pathToFile);

// Import function called by Spotlight
Boolean GetMetadataForFile(void *thisInterface,
                           CFMutableDictionaryRef attributes,
                           CFStringRef contentTypeUTI,
                           CFStringRef pathToFile) {
    @autoreleasepool {
        NSString *path = (__bridge NSString *)pathToFile;
        NSString *uti = (__bridge NSString *)contentTypeUTI;
        NSMutableDictionary *attrs = (__bridge NSMutableDictionary *)attributes;

        // Get AgentFS database path
        NSString *homeDir = NSHomeDirectory();
        NSString *dbPath = [homeDir stringByAppendingPathComponent:@".agentfs/agentfs.db"];

        // Check if file exists in AgentFS
        sqlite3 *db;
        if (sqlite3_open([dbPath UTF8String], &db) != SQLITE_OK) {
            return FALSE;
        }

        // Query for file metadata
        const char *sql = "SELECT f.id, f.checksum, f.size, f.created_at, f.updated_at "
                         "FROM files f WHERE f.path = ? AND f.deleted_at IS NULL";
        sqlite3_stmt *stmt;

        if (sqlite3_prepare_v2(db, sql, -1, &stmt, NULL) != SQLITE_OK) {
            sqlite3_close(db);
            return FALSE;
        }

        sqlite3_bind_text(stmt, 1, [path UTF8String], -1, SQLITE_TRANSIENT);

        if (sqlite3_step(stmt) != SQLITE_ROW) {
            sqlite3_finalize(stmt);
            sqlite3_close(db);
            return FALSE;
        }

        // Extract file metadata
        int64_t fileId = sqlite3_column_int64(stmt, 0);
        const char *checksum = (const char *)sqlite3_column_text(stmt, 1);
        int64_t size = sqlite3_column_int64(stmt, 2);

        sqlite3_finalize(stmt);

        // Get chunks for this file (for text content)
        NSMutableString *textContent = [NSMutableString string];

        const char *chunkSql = "SELECT content FROM file_chunks WHERE file_id = ? ORDER BY chunk_index";
        if (sqlite3_prepare_v2(db, chunkSql, -1, &stmt, NULL) == SQLITE_OK) {
            sqlite3_bind_int64(stmt, 1, fileId);

            while (sqlite3_step(stmt) == SQLITE_ROW) {
                const char *content = (const char *)sqlite3_column_text(stmt, 0);
                if (content) {
                    [textContent appendString:[NSString stringWithUTF8String:content]];
                    [textContent appendString:@"\n"];
                }
            }
            sqlite3_finalize(stmt);
        }

        sqlite3_close(db);

        // Set Spotlight attributes

        // Text content for full-text search
        if (textContent.length > 0) {
            attrs[(NSString *)kMDItemTextContent] = textContent;
        }

        // Custom AgentFS attributes
        if (checksum) {
            attrs[@"org_agentfs_checksum"] = [NSString stringWithUTF8String:checksum];
        }

        attrs[@"org_agentfs_file_id"] = @(fileId);
        attrs[@"org_agentfs_indexed"] = @YES;

        // Set display name
        NSString *fileName = [path lastPathComponent];
        attrs[(NSString *)kMDItemDisplayName] = fileName;
        attrs[(NSString *)kMDItemFSName] = fileName;

        // Content type
        attrs[(NSString *)kMDItemContentType] = uti;

        // File size
        attrs[(NSString *)kMDItemFSSize] = @(size);

        return TRUE;
    }
}
