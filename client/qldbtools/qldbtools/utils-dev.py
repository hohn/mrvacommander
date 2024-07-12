#* Interactive use only
# Experimental work with utils.py, to be merged into it.
if 0:
    from utils import *

    #* Data collection
    # Get the db information in list of DBInfo form
    db_base = "~/work-gh/mrva/mrva-open-source-download/"
    dbs = list(collect_dbs(db_base))

    # XX: add metadata
    # codeql, meta = extract_metadata('path_to_your_zipfile.zip')
    # print(codeql)
    # print(meta)

    # Inspect:
    from pprint import pprint
    pprint(["len", len(dbs)])
    pprint(["dbs[0]", dbs[0].__dict__])
    # 
    # Get a dataframe
    dbdf = pd.DataFrame([d.__dict__ for d in dbs])
    # 
    # XX: save to disk, continue use in separate session
    #
    #     PosixPath is a problem for json and parquet:
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
    # Reload to continue work
    dbdf_1 = pd.read_csv('dbdf.csv.gz', compression='gzip')
    #
    # Consistency check:
    dbdf_1.columns == dbdf.columns
    dbmask = (dbdf_1 != dbdf)
    dbdf_1[dbmask]
    dbdf_1[dbmask].dropna(how='all')
    # ctime_raw is different in places, so don't use it.
    
    # 
    # Interact with/visualize the dataframe
    os.environ['APPDATA'] = "needed-for-pandasgui"
    from pandasgui import show
    show(dbdf)
    show(cmp)
    # 
    import dtale
    dtale.show(dbdf)
    # 

# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:


import pandas as pd

# Example large DataFrame
data = {
    'name': ['Alice', 'Bob', 'Charlie', 'David', 'Eve'],
    'age': [25, 30, 35, 40, 22],
    'city': ['New York', 'Los Angeles', 'Chicago', 'Houston', 'Phoenix']
}
large_df = pd.DataFrame(data)

# Create a boolean mask: select rows where age is greater than 30
mask = large_df['age'] > 30

# Apply the boolean mask to get the smaller DataFrame
small_df = large_df[mask]

print(small_df)
