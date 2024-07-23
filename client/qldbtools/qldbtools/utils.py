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

def log_and_raise_e(message, exception):
    logging.error(message)
    raise exception(message)

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
    # Files may not be zip files:
    # {"message":"Repository was archived so is read-only.",
    # "documentation_url":"https://docs.github.com/rest/code-scanning/code-scanning#get-a-codeql-database-for-a-repository"}
    # 
    try:
        with zipfile.ZipFile(zipfile_path, 'r') as z:
            for file_info in z.infolist():
                # Filenames seen
                #       java/codeql-database.yml
                #       codeql_db/codeql-database.yml
                if file_info.filename.endswith('codeql-database.yml'):
                    with z.open(file_info) as f:
                        codeql_content = yaml.safe_load(f)
                # And
                #       java/baseline-info.json
                #       codeql_db/baseline-info.json
                elif file_info.filename.endswith('baseline-info.json'):
                    with z.open(file_info) as f:
                        meta_content = json.load(f)
    except zipfile.BadZipFile:
        log_and_raise_e(f"Not a zipfile: '{zipfile_path}'", ExtractNotZipfile)
    # The baseline-info is only available in more recent CodeQL versions
    if not meta_content:
        meta_content = {'languages':
                        {'no-language': {'displayName': 'no-language',
                                 'files': [],
                                 'linesOfCode': -1,
                                 'name': 'nolang'},
                         }}
            
    if not codeql_content:
        log_and_raise_e(f"Not a zipfile: '{zipfile_path}'", ExtractNoCQLDB)
    return codeql_content, meta_content
               
class ExtractNotZipfile(Exception): pass
class ExtractNoCQLDB(Exception): pass

#    metadata_details(codeql_content, meta_content)
#
# Extract the details from metadata that will be used in DB selection and return a
# dataframe with the information.  Example, cropped to fit:
#
# full_df.T
# Out[535]: 
#                                      0                  1
# left_index                           0                  0
# baselineLinesOfCode              17990              17990
# primaryLanguage                    cpp                cpp
# sha                  288920efc079766f4  282c20efc079766f4
# cliVersion                      2.17.0             2.17.0
# creationTime             .325253+00:00    51.325253+00:00
# finalised                         True               True
# db_lang                            cpp             python
# db_lang_displayName              C/C++             Python
# db_lang_file_count                 102                 27
# db_lang_linesOfCode              17990               5586
#
def metadata_details(left_index, codeql_content, meta_content):
    cqlc, metac = codeql_content, meta_content
    d = {'left_index': left_index,
         'baselineLinesOfCode': cqlc['baselineLinesOfCode'],
         'primaryLanguage': cqlc['primaryLanguage'],
         'sha': cqlc['creationMetadata'].get('sha', 'abcde0123'),
         'cliVersion': cqlc['creationMetadata']['cliVersion'],
         'creationTime': cqlc['creationMetadata']['creationTime'],
         'finalised': cqlc.get('finalised', pd.NA),
         }
    f = pd.DataFrame(d, index=[0])
    joiners = []
    if not ('languages' in metac):
        log_and_raise_e("Missing 'languages' in metadata", DetailsMissing)
    for lang, lang_cont in metac['languages'].items():
        d1 = { 'left_index' : left_index, 
               'db_lang':  lang }
        for prop, val in lang_cont.items():
            if prop == 'files':
                d1['db_lang_file_count'] = len(val)
            elif prop == 'linesOfCode':
                d1['db_lang_linesOfCode'] = val
            elif prop == 'displayName':
                d1['db_lang_displayName'] = val
        joiners.append(d1)
    fj = pd.DataFrame(joiners)
    full_df = pd.merge(f, fj, on='left_index', how='outer')
    return full_df

class DetailsMissing(Exception): pass                        


# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
