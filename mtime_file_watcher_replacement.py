#!/usr/bin/env python
#
# Replacement `MtimeFileWatcher` for App Engine SDK's dev_appserver.py,
# designed for OS X. Improves upon existing file watcher (under OS X) in
# numerous ways:
#
#   - Uses FSEvents API to watch for changes instead of polling. This saves a
#     dramatic amount of CPU, especially in projects with several modules.
#   - Tries to be smarter about which modules reload when files change, only
#     modified module should reload.
#
import logging
import os
import time
from ConfigParser import ConfigParser, NoSectionError

from watchdog.events import FileSystemEventHandler, PatternMatchingEventHandler, FileSystemEvent
from watchdog.observers import Observer

# Only watch for changes to .go, .py or .yaml files
WATCHED_EXTENSIONS = {'.go', '.py', '.yaml'}


def find_upwards(file_name, start_at=os.getcwd()):
    cur_dir = start_at
    while True:
        file_list = os.listdir(cur_dir)
        parent_dir = os.path.dirname(cur_dir)
        if file_name in file_list:
            return cur_dir
        else:
            if cur_dir == parent_dir:
                return None
            else:
                cur_dir = parent_dir


class MtimeFileWatcher(object):
    SUPPORTS_MULTIPLE_DIRECTORIES = True

    def __init__(self, directories, **kwargs):
        self._changes = _changes = []
        # Path to current module
        module_dir = directories[0]

        watched_extensions = WATCHED_EXTENSIONS
        setup_cfg_path = find_upwards("setup.cfg")
        if setup_cfg_path:
            config = ConfigParser()
            try:
                config_value = config.get('appengine:mtime_file_watcher', 'watched_extensions')
            except NoSectionError:
                watched_extensions = WATCHED_EXTENSIONS
            else:
                try:
                    watched_extensions = set(config_value)
                except TypeError:
                    watched_extensions = WATCHED_EXTENSIONS

        # pattern matching files with extension
        patterns = ["*{}".format(ext) for ext in watched_extensions]
        logging.info("Stating fs watching on {} with pattern:{}".format(module_dir, patterns))
        event_handler = MyEventHandler(patterns, _changes)
        self.observer = Observer()
        self.observer.schedule(event_handler, module_dir, recursive=True)

    def start(self):
        self.observer.start()

    def changes(self, timeout=None):
        time.sleep(1)
        changed = set(self._changes)
        del self._changes[:]
        return changed

    def quit(self):
        try:
            self.observer.stop()
        except Exception:
            pass


class MyEventHandler(PatternMatchingEventHandler):

    def __init__(self, patterns, changes=None):
        PatternMatchingEventHandler.__init__(self, patterns=patterns)
        self._changes = changes

    def on_any_event(self, event):
        self._changes.append(event.src_path)
