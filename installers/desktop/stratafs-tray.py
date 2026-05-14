#!/usr/bin/env python3
"""
StrataFS System Tray Application
A cross-platform system tray interface for StrataFS
"""

import sys
import os
import subprocess
import time
import threading
import webbrowser
import json
from pathlib import Path

try:
    import tkinter as tk
    from tkinter import messagebox, simpledialog
    HAS_TKINTER = True
except ImportError:
    HAS_TKINTER = False

try:
    import pystray
    from PIL import Image, ImageDraw
    HAS_PYSTRAY = True
except ImportError:
    HAS_PYSTRAY = False

class StrataFSTray:
    def __init__(self):
        self.config_dir = Path.home() / '.stratafs'
        self.binary_name = 'stratafs'
        self.api_port = 8080
        self.mcp_port = 8081
        self.process = None
        self.icon = None
        self.running = False

        # Load configuration
        self.load_config()

        # Create tray icon
        if HAS_PYSTRAY:
            self.create_tray_icon()
        elif HAS_TKINTER:
            self.create_tkinter_gui()
        else:
            print("Neither pystray nor tkinter available. Running in console mode.")
            self.console_mode()

    def load_config(self):
        """Load StrataFS configuration"""
        config_file = self.config_dir / 'config.json'
        if config_file.exists():
            try:
                with open(config_file, 'r') as f:
                    config = json.load(f)
                    self.api_port = config.get('server', {}).get('api_port', 8080)
                    self.mcp_port = config.get('server', {}).get('mcp_port', 8081)
            except Exception as e:
                print(f"Error loading config: {e}")

    def create_tray_icon(self):
        """Create system tray icon using pystray"""
        # Create icon image
        image = self.create_icon_image()

        # Create menu
        menu = pystray.Menu(
            pystray.MenuItem("StrataFS", None, enabled=False),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("Start", self.start_stratafs, enabled=lambda item: not self.is_running()),
            pystray.MenuItem("Stop", self.stop_stratafs, enabled=lambda item: self.is_running()),
            pystray.MenuItem("Restart", self.restart_stratafs),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("Web Interface", self.open_web_interface, enabled=lambda item: self.is_running()),
            pystray.MenuItem("Status", self.show_status),
            pystray.MenuItem("Configuration", self.show_config),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("Initialize Config", self.init_config),
            pystray.MenuItem("Add Source", self.add_source),
            pystray.Menu.SEPARATOR,
            pystray.MenuItem("Exit", self.quit_application)
        )

        # Create tray icon
        self.icon = pystray.Icon("stratafs", image, "StrataFS", menu)

        # Start monitoring in background
        self.start_monitoring()

        # Run the icon
        self.icon.run()

    def create_tkinter_gui(self):
        """Create simple GUI using tkinter"""
        root = tk.Tk()
        root.title("StrataFS Control")
        root.geometry("300x400")

        # Status label
        self.status_label = tk.Label(root, text="Status: Checking...", font=("Arial", 12, "bold"))
        self.status_label.pack(pady=10)

        # Buttons
        button_frame = tk.Frame(root)
        button_frame.pack(pady=10)

        tk.Button(button_frame, text="Start", command=self.start_stratafs, width=15).pack(pady=5)
        tk.Button(button_frame, text="Stop", command=self.stop_stratafs, width=15).pack(pady=5)
        tk.Button(button_frame, text="Restart", command=self.restart_stratafs, width=15).pack(pady=5)

        tk.Frame(root, height=2, bg="gray").pack(fill="x", pady=10)

        tk.Button(root, text="Web Interface", command=self.open_web_interface, width=15).pack(pady=5)
        tk.Button(root, text="Status", command=self.show_status, width=15).pack(pady=5)
        tk.Button(root, text="Configuration", command=self.show_config, width=15).pack(pady=5)

        tk.Frame(root, height=2, bg="gray").pack(fill="x", pady=10)

        tk.Button(root, text="Initialize Config", command=self.init_config, width=15).pack(pady=5)
        tk.Button(root, text="Add Source", command=self.add_source, width=15).pack(pady=5)

        # Start monitoring
        self.start_monitoring()
        self.update_status_label()

        root.mainloop()

    def console_mode(self):
        """Run in console mode"""
        print("StrataFS Console Controller")
        print("Commands: start, stop, restart, status, config, init, add, web, quit")

        while True:
            try:
                cmd = input("\nStrataFS> ").strip().lower()
                if cmd == "start":
                    self.start_stratafs()
                elif cmd == "stop":
                    self.stop_stratafs()
                elif cmd == "restart":
                    self.restart_stratafs()
                elif cmd == "status":
                    self.show_status()
                elif cmd == "config":
                    self.show_config()
                elif cmd == "init":
                    self.init_config()
                elif cmd == "add":
                    self.add_source()
                elif cmd == "web":
                    self.open_web_interface()
                elif cmd in ["quit", "exit", "q"]:
                    break
                else:
                    print("Unknown command. Available: start, stop, restart, status, config, init, add, web, quit")
            except KeyboardInterrupt:
                break
            except EOFError:
                break

    def create_icon_image(self):
        """Create icon image"""
        # Create a simple icon
        width = 64
        height = 64
        color = "blue" if self.is_running() else "gray"

        image = Image.new('RGB', (width, height), color="white")
        dc = ImageDraw.Draw(image)

        # Draw a simple "A" for StrataFS
        dc.rectangle([10, 10, width-10, height-10], fill=color)
        dc.text((width//2-8, height//2-8), "A", fill="white")

        return image

    def start_monitoring(self):
        """Start background monitoring thread"""
        def monitor():
            while True:
                time.sleep(5)
                if HAS_PYSTRAY and self.icon:
                    # Update icon
                    self.icon.icon = self.create_icon_image()

        thread = threading.Thread(target=monitor, daemon=True)
        thread.start()

    def update_status_label(self):
        """Update status label in tkinter GUI"""
        if hasattr(self, 'status_label'):
            status = "RUNNING" if self.is_running() else "STOPPED"
            self.status_label.config(text=f"Status: {status}")
            # Schedule next update
            self.status_label.after(5000, self.update_status_label)

    def is_running(self):
        """Check if StrataFS is running"""
        try:
            if sys.platform == "win32":
                result = subprocess.run(['tasklist', '/FI', 'IMAGENAME eq stratafs.exe'],
                                      capture_output=True, text=True)
                return 'stratafs.exe' in result.stdout
            else:
                result = subprocess.run(['pgrep', '-f', self.binary_name],
                                      capture_output=True)
                return result.returncode == 0
        except:
            return False

    def start_stratafs(self, item=None):
        """Start StrataFS"""
        if self.is_running():
            self.show_message("StrataFS is already running")
            return

        try:
            # Initialize config if needed
            if not (self.config_dir / 'config.json').exists():
                self.init_config()

            # Start StrataFS
            if sys.platform == "win32":
                subprocess.Popen([self.binary_name, '--config-dir', str(self.config_dir)],
                               creationflags=subprocess.CREATE_NO_WINDOW)
            else:
                subprocess.Popen([self.binary_name, '--config-dir', str(self.config_dir)],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

            # Wait a moment and check
            time.sleep(2)
            if self.is_running():
                self.show_message("StrataFS started successfully!")
            else:
                self.show_message("Failed to start StrataFS", error=True)

        except Exception as e:
            self.show_message(f"Error starting StrataFS: {e}", error=True)

    def stop_stratafs(self, item=None):
        """Stop StrataFS"""
        if not self.is_running():
            self.show_message("StrataFS is not running")
            return

        try:
            if sys.platform == "win32":
                subprocess.run(['taskkill', '/F', '/IM', 'stratafs.exe'])
            else:
                subprocess.run(['pkill', '-f', self.binary_name])

            # Wait a moment and check
            time.sleep(2)
            if not self.is_running():
                self.show_message("StrataFS stopped successfully!")
            else:
                self.show_message("Failed to stop StrataFS", error=True)

        except Exception as e:
            self.show_message(f"Error stopping StrataFS: {e}", error=True)

    def restart_stratafs(self, item=None):
        """Restart StrataFS"""
        self.stop_stratafs()
        time.sleep(2)
        self.start_stratafs()

    def open_web_interface(self, item=None):
        """Open web interface in browser"""
        if not self.is_running():
            if self.confirm_message("StrataFS is not running. Start it?"):
                self.start_stratafs()
                time.sleep(3)  # Wait for startup
            else:
                return

        url = f"http://localhost:{self.api_port}"
        webbrowser.open(url)

    def show_status(self, item=None):
        """Show StrataFS status"""
        if self.is_running():
            try:
                if sys.platform == "win32":
                    result = subprocess.run(['tasklist', '/FI', 'IMAGENAME eq stratafs.exe', '/FO', 'CSV'],
                                          capture_output=True, text=True)
                    lines = result.stdout.strip().split('\n')
                    if len(lines) > 1:
                        pid = lines[1].split(',')[1].strip('"')
                    else:
                        pid = "Unknown"
                else:
                    result = subprocess.run(['pgrep', '-f', self.binary_name],
                                          capture_output=True, text=True)
                    pid = result.stdout.strip().split('\n')[0] if result.stdout.strip() else "Unknown"

                message = f"""StrataFS Status: RUNNING
PID: {pid}

Web Interface: http://localhost:{self.api_port}
MCP Server: http://localhost:{self.mcp_port}

Log file: {self.config_dir}/desktop.log"""
            except:
                message = "StrataFS Status: RUNNING (details unavailable)"
        else:
            message = """StrataFS Status: STOPPED

Web Interface: Not available
MCP Server: Not available"""

        self.show_message(message)

    def show_config(self, item=None):
        """Show configuration"""
        config_file = self.config_dir / 'config.json'
        if config_file.exists():
            if sys.platform == "win32":
                os.startfile(str(config_file))
            elif sys.platform == "darwin":
                subprocess.run(['open', str(config_file)])
            else:
                subprocess.run(['xdg-open', str(config_file)])
        else:
            self.show_message("Configuration file not found. Please initialize StrataFS first.", error=True)

    def init_config(self, item=None):
        """Initialize configuration"""
        try:
            subprocess.run([self.binary_name, 'config', 'init', '--config-dir', str(self.config_dir)],
                         check=True)
            self.show_message("Configuration initialized successfully!")
            self.load_config()  # Reload config
        except Exception as e:
            self.show_message(f"Error initializing configuration: {e}", error=True)

    def add_source(self, item=None):
        """Add a storage source"""
        if HAS_TKINTER:
            # Use tkinter dialog
            root = tk.Tk()
            root.withdraw()
            path = simpledialog.askstring("Add Source", "Enter directory path to index:")
            root.destroy()

            if path:
                try:
                    # This would need to be implemented in the StrataFS binary
                    result = subprocess.run([self.binary_name, 'source', 'add', path,
                                           '--config-dir', str(self.config_dir)],
                                          capture_output=True, text=True)
                    if result.returncode == 0:
                        self.show_message(f"Source added successfully: {path}")
                    else:
                        self.show_message(f"Error adding source: {result.stderr}", error=True)
                except Exception as e:
                    self.show_message(f"Error adding source: {e}", error=True)
        else:
            self.show_message("Please use the command line to add sources:\nstratafs source add <path>")

    def show_message(self, message, error=False):
        """Show message to user"""
        if HAS_TKINTER:
            if error:
                messagebox.showerror("StrataFS", message)
            else:
                messagebox.showinfo("StrataFS", message)
        else:
            print(f"{'ERROR' if error else 'INFO'}: {message}")

    def confirm_message(self, message):
        """Show confirmation dialog"""
        if HAS_TKINTER:
            return messagebox.askyesno("StrataFS", message)
        else:
            response = input(f"{message} [y/N]: ").strip().lower()
            return response in ['y', 'yes']

    def quit_application(self, item=None):
        """Quit the application"""
        if self.is_running():
            if self.confirm_message("StrataFS is running. Stop it before exiting?"):
                self.stop_stratafs()

        if HAS_PYSTRAY and self.icon:
            self.icon.stop()

        sys.exit(0)

def main():
    """Main entry point"""
    if len(sys.argv) > 1:
        # Command line mode
        tray = StrataFSTray()
        command = sys.argv[1].lower()

        if command == "start":
            tray.start_stratafs()
        elif command == "stop":
            tray.stop_stratafs()
        elif command == "restart":
            tray.restart_stratafs()
        elif command == "status":
            tray.show_status()
        elif command == "web":
            tray.open_web_interface()
        elif command == "init":
            tray.init_config()
        else:
            print(f"Unknown command: {command}")
            print("Available commands: start, stop, restart, status, web, init")
    else:
        # GUI mode
        tray = StrataFSTray()

if __name__ == "__main__":
    main()