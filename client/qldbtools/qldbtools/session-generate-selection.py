""" Read a table of CodeQL DB information
    and generate the selection files for
    1. the VS Code CodeQL plugin
    2. the gh-mrva command-line client
"""
#
#* Collect the information and write files
#
import pandas as pd
import sys
import qldbtools.utils as utils
import numpy as np
import importlib
importlib.reload(utils)

df0 = pd.read_csv('scratch/db-info-3.csv')

# Use num_entries, chosen via pseudo-random numbers
df1 = df0.sample(n=3, random_state=np.random.RandomState(4242))

repos = []
for index, row in df1[['owner', 'name', 'CID', 'path']].iterrows():
    owner, name, CID, path = row
    repos.append(utils.form_db_req_name(owner, name, CID))

repo_list_name = "mirva-list"
vsc = {
    "version": 1,
    "databases": {
        "variantAnalysis": {
            "repositoryLists": [
                {
                    "name": repo_list_name,
                    "repositories": repos,
                }
            ],
            "owners": [],
            "repositories": []
        }
    },
    "selected": {
        "kind": "variantAnalysisUserDefinedList",
        "listName": repo_list_name
    }
}

gh = {
    repo_list_name:  repos
}


# write the files
import json
with open("tmp-selection-vsc.json", "w") as fc:
    json.dump(vsc, fc, indent=4)
with open("tmp-selection-gh.json", "w") as fc:
    json.dump(gh, fc, indent=4)
    
# Local Variables:
# python-shell-virtualenv-root: "~/work-gh/mrva/mrvacommander/client/qldbtools/venv/"
# End:
