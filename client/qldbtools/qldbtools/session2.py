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

#
#* Collect metadata from DB zip files
#
d = dbdf_1
d.path[0]
idb, ibl = extract_metadata(d.path[0])

idb['baselineLinesOfCode']
idb['primaryLanguage']
idb['creationMetadata']['sha']
idb['creationMetadata']['cliVersion']
idb['creationMetadata']['creationTime'].isoformat()
idb['finalised']

for lang, lang_cont in ibl['languages'].items():
    print(lang)
    indent = "    "
    for prop, val in lang_cont.items():
        if prop == 'files':
            print("%sfiles count %d" % (indent, len(val)))
        elif prop == 'linesOfCode':
            print("%slinesOfCode %d" % (indent, val))
        elif prop == 'displayName':
            print("%sdisplayName %s" % (indent, val))


# 
# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
# 
