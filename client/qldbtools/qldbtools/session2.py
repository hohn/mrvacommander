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
#** A manual sample
#
d = dbdf_1
left_index = 0
d.path[0]
cqlc, metac = extract_metadata(d.path[0])

cqlc['baselineLinesOfCode']
cqlc['primaryLanguage']
cqlc['creationMetadata']['sha']
cqlc['creationMetadata']['cliVersion']
cqlc['creationMetadata']['creationTime'].isoformat()
cqlc['finalised']

for lang, lang_cont in metac['languages'].items():
    print(lang)
    indent = "    "
    for prop, val in lang_cont.items():
        if prop == 'files':
            print("%sfiles count %d" % (indent, len(val)))
        elif prop == 'linesOfCode':
            print("%slinesOfCode %d" % (indent, val))
        elif prop == 'displayName':
            print("%sdisplayName %s" % (indent, val))

#** Automated for all entries
# The rest of this interactive script is available as cli script in
# mc-db-refine-info
d = dbdf_1
joiners = []
for left_index in range(0, len(d)-1):
    try:
        cqlc, metac = extract_metadata(d.path[left_index])
    except ExtractNotZipfile:
        continue
    except ExtractNoCQLDB:
        continue
    try:
        detail_df = metadata_details(left_index, cqlc, metac)
    except DetailsMissing:
        continue
    joiners.append(detail_df)
joiners_df = pd.concat(joiners, axis=0)
full_df = pd.merge(d, joiners_df, left_index=True, right_on='left_index', how='outer')    

#** View the full dataframe with metadata
from pandasgui import show
os.environ['APPDATA'] = "needed-for-pandasgui"
show(full_df)

#** Re-order the dataframe columns by importance
# - Much of the data
#   1. Is only conditionally present
#   2. Is extra info, not for the DB proper
#   3. May have various names

# - The essential columns are
#     | owner               |
#     | name                |
#     | language            |
#     | size                |
#     | cliVersion          |
#     | creationTime        |
#     | sha                 |
#     | baselineLinesOfCode |
#     | path                |

# - The rest are useful; put them last
#     | db_lang             |
#     | db_lang_displayName |
#     | db_lang_file_count  |
#     | db_lang_linesOfCode |
#     | left_index          |
#     | ctime               |
#     | primaryLanguage     |
#     | finalised           |

final_df = full_df.reindex(columns=['owner', 'name', 'language', 'size', 'cliVersion',
	                                'creationTime', 'sha', 'baselineLinesOfCode', 'path',
	                                'db_lang', 'db_lang_displayName', 'db_lang_file_count',
	                                'db_lang_linesOfCode', 'ctime', 'primaryLanguage',
	                                'finalised', 'left_index'])

final_df.to_csv('all-info-table.csv.gz', compression='gzip', index=False)

# 
# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
# 
