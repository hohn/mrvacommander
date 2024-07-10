#* Overview
# - [ ] import the dbs from the filesystem.  Include
#   1. name
#   2. owner
#   3. language
#   4. creation date
#   5. db size
#* Imports 
import pandas as pd
from pathlib import Path
import os
import logging
import time

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
class DBInfo:
    pass

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
            db.ctime_raw = s.st_ctime
            db.ctime = time.ctime(s.st_ctime)
            yield db 

def dbdf_from_tree():
    db_base = "~/work-gh/mrva/mrva-open-source-download/"
    dbs = list(collect_dbs(db_base))
    dbdf = pd.DataFrame([d.__dict__ for d in dbs])
    return dbdf
    
#* Interactive use only
if 0:
    #* Data collection
    # Get the db information in list of DBInfo form
    db_base = "~/work-gh/mrva/mrva-open-source-download/"
    dbs = list(collect_dbs(db_base))
    # 
    # Inspect:
    from pprint import pprint
    pprint(["len", len(dbs)])
    pprint(["dbs[0]", dbs[0].__dict__])
    # 
    # Get a dataframe
    dbdf = pd.DataFrame([d.__dict__ for d in dbs])
    # 
    # Interact with/visualize it
    os.environ['APPDATA'] = "needed-for-pandasgui"
    from pandasgui import show
    show(dbdf)
    # 
    import dtale
    dtale.show(dbdf)
    # 

# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/venv/"
# End:
