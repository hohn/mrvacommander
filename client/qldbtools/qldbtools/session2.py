# Experimental work with utils.py, to be merged into it.
from utils import *
from pprint import pprint

#* Reload gzipped CSV file to continue work
dbdf_1 = pd.read_csv('dbdf.csv.gz', compression='gzip')
#
# (old) Consistency check:
# dbdf_1.columns == dbdf.columns
# dbmask = (dbdf_1 != dbdf)
# dbdf_1[dbmask]
# dbdf_1[dbmask].dropna(how='all')
# ctime_raw is different in places, so don't use it.

# 
#* Interact with/visualize the dataframe
# Using pandasgui -- qt
from pandasgui import show
os.environ['APPDATA'] = "needed-for-pandasgui"
show(dbdf_1)
# Using dtale -- web
import dtale
dtale.show(dbdf_1)
# 

# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
