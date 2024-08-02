# Experimental work for ../bin/mc-db-unique, to be merged into it.
import qldbtools.utils as utils
from pprint import pprint
import pandas as pd
# cd ../

#* Reload CSV file to continue work
df2 = df_refined = pd.read_csv('scratch/db-info-2.csv')

# Identify rows missing specific entries
rows = ( df2['cliVersion'].isna() | 
         df2['creationTime'].isna() |
         df2['language'].isna() |
         df2['sha'].isna() )
df2[rows]
df3 = df2[~rows]
df3

#* post-save work
df4 = pd.read_csv('scratch/db-info-3.csv')

# Sort and group
df_sorted = df4.sort_values(by=['owner', 'name', 'CID', 'creationTime'])
df_unique = df_sorted.groupby(['owner', 'name', 'CID']).first().reset_index()

# Find duplicates
df_dups = df_unique[df_unique['CID'].duplicated(keep=False)]
len(df_dups)
df_dups['CID']

# Set display options
pd.set_option('display.max_colwidth', None)
pd.set_option('display.max_columns', None)
pd.set_option('display.width', 140)


# 
# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
# 
