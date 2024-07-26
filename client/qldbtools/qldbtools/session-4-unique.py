# Experimental work with utils.py, to be merged into it.
from utils import *
from pprint import pprint

#* Reload gzipped CSV file to continue work
df2 = pd.read_csv('db-info-2.csv')


df_sorted = df2.sort_values(by=['owner', 'name', 'creationTime'])
df_unique = df_sorted.groupby(['owner', 'name']).first().reset_index()

# 
# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
# 
