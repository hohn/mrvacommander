#* Overview
# - [ ] import the dbs from the filesystem.  Include
#   1. name
#   2. owner
#   3. language
#   4. creation date
#   5. db size
#* Imports 
from dataclasses import dataclass
from pathlib import Path

import datetime
import json
import logging
import os
import pandas as pd
import time
import yaml
import zipfile

#* Setup
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)

#* Utility functions
def log_and_raise(message):
    logging.error(message)
    raise Exception(message)

def traverse_tree(root):
    root_path = Path(os.path.expanduser(root))
    if not root_path.exists() or not root_path.is_dir():
        log_and_raise(f"The specified root path '{root}' does not exist or "
                      "is not a directory.")
    for path in root_path.rglob('*'):
        if path.is_file():
            yield path
        elif path.is_dir():
            pass

# Collect information in one 'struct'
@dataclass
class DBInfo:
    ctime : str = '2024-05-13T12:04:01.593586'
    language : str = 'cpp'
    name : str = 'nanobind'
    owner : str = 'wjakob'
    path : Path = Path('/Users/hohn/work-gh/mrva/mrva-open-source-download/repos/wjakob/nanobind/code-scanning/codeql/databases/cpp/db.zip')
    size : int = 63083064


def collect_dbs(db_base):
    for path in traverse_tree(db_base):
        if path.name == "db.zip":
            # For the current repository, we have
            #     In [292]: len(path.parts)
            #     Out[292]: 14
            # and can work from the end to get relevant info from the file path.
            db = DBInfo()
            (*_, db.owner, db.name, _, _, _, db.language, _) = path.parts
            db.path = path
            s = path.stat()
            db.size = s.st_size
            # db.ctime_raw = s.st_ctime
            # db.ctime = time.ctime(s.st_ctime)
            db.ctime = datetime.datetime.fromtimestamp(s.st_ctime).isoformat()
            yield db 

def dbdf_from_tree():
    db_base = "~/work-gh/mrva/mrva-open-source-download/"
    dbs = list(collect_dbs(db_base))
    dbdf = pd.DataFrame([d.__dict__ for d in dbs])
    return dbdf
    
#    extract_metadata(zipfile)
# 
# Unzip zipfile into memory and return the contents of the files
# codeql-database.yml and baseline-info.json that it contains in a tuple
#
def extract_metadata(zipfile_path):
    codeql_content = None
    meta_content = None
    with zipfile.ZipFile(zipfile_path, 'r') as z:
        for file_info in z.infolist():
            if file_info.filename == 'codeql_db/codeql-database.yml':
                with z.open(file_info) as f:
                    codeql_content = yaml.safe_load(f)
            elif file_info.filename == 'codeql_db/baseline-info.json':
                with z.open(file_info) as f:
                    meta_content = json.load(f)
    return codeql_content, meta_content
               
# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
