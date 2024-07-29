#* Experimental work with utils.py, to be merged into it.
# The rest of this interactive script is available as cli script in
# mc-db-initial-info
from utils import *

#* Data collection
# Get the db information in list of DBInfo form
db_base = "~/work-gh/mrva/mrva-open-source-download/"
dbs = list(collect_dbs(db_base))

# Inspect:
from pprint import pprint
pprint(["len", len(dbs)])
pprint(["dbs[0]", dbs[0].__dict__])
pprint(["dbs[-1]", dbs[-1].__dict__])
# 
# Get a dataframe
dbdf = pd.DataFrame([d.__dict__ for d in dbs])
# 
#* Experiments with on-disk format
# Continue use of raw information in separate session.
# 
# PosixPath is a problem for json and parquet
# 
dbdf['path'] = dbdf['path'].astype(str)
#
dbdf.to_csv('dbdf.csv')
#
dbdf.to_csv('dbdf.csv.gz', compression='gzip', index=False)
# 
dbdf.to_json('dbdf.json')
#
# dbdf.to_hdf('dbdf.h5', key='dbdf', mode='w')
# 
# fast, binary
dbdf.to_parquet('dbdf.parquet')
# 
# fast
import sqlite3
conn = sqlite3.connect('dbdf.db')
dbdf.to_sql('qldbs', conn, if_exists='replace', index=False)
conn.close()
# 
# Sizes:
# ls -laSr dbdf.*
# -rw-r--r--@ 1 hohn  staff  101390 Jul 12 14:17 dbdf.csv.gz
# -rw-r--r--@ 1 hohn  staff  202712 Jul 12 14:17 dbdf.parquet
# -rw-r--r--@ 1 hohn  staff  560623 Jul 12 14:17 dbdf.csv
# -rw-r--r--@ 1 hohn  staff  610304 Jul 12 14:17 dbdf.db
# -rw-r--r--@ 1 hohn  staff  735097 Jul 12 14:17 dbdf.json
#
# parquet has many libraries, including go: xitongsys/parquet-go
# https://parquet.apache.org/
# 


# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
