#!/usr/bin/env python3
"""
AgentFS Nautilus Extension
Adds context menu items for AgentFS operations in GNOME Files.

Install to: ~/.local/share/nautilus-python/extensions/
"""

import os
import subprocess
import json
from urllib.parse import quote

from gi.repository import Nautilus, GObject, Gio, GLib

class AgentFSExtension(GObject.GObject, Nautilus.MenuProvider):
    """Nautilus extension for AgentFS file operations."""

    def __init__(self):
        super().__init__()
        self.api_url = os.environ.get('AGENTFS_API_URL', 'http://localhost:8080')

    def get_file_items(self, files):
        """Return context menu items for selected files."""
        if not files:
            return []

        items = []

        # Single file actions
        if len(files) == 1:
            file_obj = files[0]
            if file_obj.get_uri_scheme() == 'file':
                filepath = file_obj.get_location().get_path()

                # View Metadata action
                item = Nautilus.MenuItem(
                    name='AgentFS::ViewMetadata',
                    label='View AgentFS Metadata',
                    tip='View semantic metadata for this file',
                    icon='document-properties'
                )
                item.connect('activate', self._on_view_metadata, filepath)
                items.append(item)

                # View Chunks action
                item = Nautilus.MenuItem(
                    name='AgentFS::ViewChunks',
                    label='View AgentFS Chunks',
                    tip='View text chunks extracted from this file',
                    icon='view-list-symbolic'
                )
                item.connect('activate', self._on_view_chunks, filepath)
                items.append(item)

                # Search Similar action
                item = Nautilus.MenuItem(
                    name='AgentFS::SearchSimilar',
                    label='Find Similar Files',
                    tip='Search for semantically similar files',
                    icon='edit-find-symbolic'
                )
                item.connect('activate', self._on_search_similar, filepath)
                items.append(item)

        # Multi-file actions
        filepaths = []
        for f in files:
            if f.get_uri_scheme() == 'file':
                filepaths.append(f.get_location().get_path())

        if filepaths:
            # Reindex action
            item = Nautilus.MenuItem(
                name='AgentFS::Reindex',
                label=f'Reindex in AgentFS ({len(filepaths)} file{"s" if len(filepaths) > 1 else ""})',
                tip='Queue files for reindexing',
                icon='view-refresh-symbolic'
            )
            item.connect('activate', self._on_reindex, filepaths)
            items.append(item)

        return items

    def get_background_items(self, current_folder):
        """Return context menu items for folder background."""
        if current_folder.get_uri_scheme() != 'file':
            return []

        folderpath = current_folder.get_location().get_path()
        items = []

        # Add source action
        item = Nautilus.MenuItem(
            name='AgentFS::AddSource',
            label='Add Folder to AgentFS',
            tip='Add this folder as an AgentFS source',
            icon='list-add-symbolic'
        )
        item.connect('activate', self._on_add_source, folderpath)
        items.append(item)

        # Export metadata action
        item = Nautilus.MenuItem(
            name='AgentFS::ExportMetadata',
            label='Export AgentFS Metadata Here',
            tip='Export metadata view to this folder',
            icon='document-save-symbolic'
        )
        item.connect('activate', self._on_export_metadata, folderpath)
        items.append(item)

        return items

    def _on_view_metadata(self, menu, filepath):
        """Show metadata for the selected file."""
        try:
            result = subprocess.run(
                ['agentfs', 'file', 'info', filepath],
                capture_output=True,
                text=True,
                timeout=10
            )
            self._show_dialog('AgentFS Metadata', result.stdout or result.stderr or 'No metadata found')
        except subprocess.TimeoutExpired:
            self._show_dialog('Error', 'Request timed out')
        except FileNotFoundError:
            self._show_dialog('Error', 'agentfs command not found. Is AgentFS installed?')
        except Exception as e:
            self._show_dialog('Error', str(e))

    def _on_view_chunks(self, menu, filepath):
        """Show chunks for the selected file."""
        try:
            result = subprocess.run(
                ['agentfs', 'file', 'chunks', filepath],
                capture_output=True,
                text=True,
                timeout=10
            )
            self._show_dialog('AgentFS Chunks', result.stdout or result.stderr or 'No chunks found')
        except subprocess.TimeoutExpired:
            self._show_dialog('Error', 'Request timed out')
        except FileNotFoundError:
            self._show_dialog('Error', 'agentfs command not found. Is AgentFS installed?')
        except Exception as e:
            self._show_dialog('Error', str(e))

    def _on_search_similar(self, menu, filepath):
        """Search for files similar to the selected file."""
        try:
            # Get first chunk content to use as search query
            result = subprocess.run(
                ['agentfs', 'file', 'chunks', filepath, '--limit', '1', '--format', 'text'],
                capture_output=True,
                text=True,
                timeout=10
            )
            if result.returncode == 0 and result.stdout:
                # Open in browser or UI
                query = result.stdout[:200]  # First 200 chars
                url = f"{self.api_url}/docs?q={quote(query)}"
                subprocess.Popen(['xdg-open', url])
            else:
                self._show_dialog('Error', 'Could not extract content for similarity search')
        except Exception as e:
            self._show_dialog('Error', str(e))

    def _on_reindex(self, menu, filepaths):
        """Queue files for reindexing."""
        try:
            for filepath in filepaths:
                subprocess.run(
                    ['agentfs', 'file', 'reindex', filepath],
                    capture_output=True,
                    timeout=10
                )
            self._show_notification(f'Queued {len(filepaths)} file(s) for reindexing')
        except subprocess.TimeoutExpired:
            self._show_dialog('Error', 'Request timed out')
        except FileNotFoundError:
            self._show_dialog('Error', 'agentfs command not found. Is AgentFS installed?')
        except Exception as e:
            self._show_dialog('Error', str(e))

    def _on_add_source(self, menu, folderpath):
        """Add folder as an AgentFS source."""
        try:
            result = subprocess.run(
                ['agentfs', 'source', 'add', folderpath],
                capture_output=True,
                text=True,
                timeout=10
            )
            if result.returncode == 0:
                self._show_notification(f'Added source: {folderpath}')
            else:
                self._show_dialog('Error', result.stderr or 'Failed to add source')
        except subprocess.TimeoutExpired:
            self._show_dialog('Error', 'Request timed out')
        except FileNotFoundError:
            self._show_dialog('Error', 'agentfs command not found. Is AgentFS installed?')
        except Exception as e:
            self._show_dialog('Error', str(e))

    def _on_export_metadata(self, menu, folderpath):
        """Export metadata to the selected folder."""
        try:
            result = subprocess.run(
                ['agentfs', 'fs', 'export', '--output', folderpath],
                capture_output=True,
                text=True,
                timeout=60
            )
            if result.returncode == 0:
                self._show_notification(f'Exported metadata to: {folderpath}')
            else:
                self._show_dialog('Error', result.stderr or 'Failed to export metadata')
        except subprocess.TimeoutExpired:
            self._show_dialog('Error', 'Export timed out')
        except FileNotFoundError:
            self._show_dialog('Error', 'agentfs command not found. Is AgentFS installed?')
        except Exception as e:
            self._show_dialog('Error', str(e))

    def _show_dialog(self, title, message):
        """Show a dialog with the given message."""
        subprocess.Popen([
            'zenity', '--info',
            '--title', title,
            '--text', message,
            '--width', '500'
        ])

    def _show_notification(self, message):
        """Show a desktop notification."""
        subprocess.Popen([
            'notify-send',
            'AgentFS',
            message,
            '-i', 'document-properties'
        ])
