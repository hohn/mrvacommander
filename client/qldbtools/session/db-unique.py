# Session around bin/mc-db-unique
import qldbtools.utils as utils
import pandas as pd

#
#* Collect the information
#
df1 = pd.read_csv("scratch/db-info-2.csv")

# Add single uniqueness field -- CID (Cumulative ID) -- using
# - creationTime
# - sha
# - cliVersion
# - language

from hashlib import blake2b

def cid_hash(row_tuple: tuple):
    """
        cid_hash(row_tuple)
    Take a bytes object and return hash as hex string
    """
    h = blake2b(digest_size = 3)
    h.update(str(row_tuple).encode())
    # return int.from_bytes(h.digest(), byteorder='big')
    return h.hexdigest()

# Apply the cid_hash function to the specified columns and create the 'CID' column
df1['CID'] = df1.apply(lambda row: cid_hash( (row['creationTime'],
                                              row['sha'], 
                                              row['cliVersion'], 
                                              row['language'])
                                            ), axis=1)

df2 = df1.reindex(columns=['owner', 'name', 'cliVersion', 'creationTime',
	                       'language', 'sha','CID', 'baselineLinesOfCode', 'path',
	                       'db_lang', 'db_lang_displayName', 'db_lang_file_count',
	                       'db_lang_linesOfCode', 'ctime', 'primaryLanguage',
	                       'finalised', 'left_index', 'size'])

df1['cid']


# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
