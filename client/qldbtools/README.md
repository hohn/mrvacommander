# qldbtools

qldbtools is a Python package for working with CodeQL databases

## Installation

-   Set up the virtual environment and install tools

                cd ~/work-gh/mrva/mrvacommander/client/qldbtools/
                python3.11 -m venv venv
                source venv/bin/activate
                pip install --upgrade pip

                # From requirements.txt
                pip install -r requirements.txt
                # Or explicitly
                pip install jupyterlab pandas ipython
                pip install lckr-jupyterlab-variableinspector

-   Run jupyterlab

                cd ~/work-gh/mrva/mrvacommander/client
                source venv/bin/activate
                jupyter lab &
               
        The variable inspector is a right-click on an open console or notebook.
       
        The `jupyter` command produces output including
       
                Jupyter Server 2.14.1 is running at:
                http://127.0.0.1:8888/lab?token=4c91308819786fe00a33b76e60f3321840283486457516a1

        Use this to connect multiple front ends

-   Local development

        ```bash
        cd ~/work-gh/mrva/mrvacommander/client/qldbtools
        source venv/bin/activate
        pip install --editable .
        ```

        The `--editable` *should* use symlinks for all scripts; use `./bin/*` to be sure.


-   Full installation

        ```bash
        pip install qldbtools
        ```


## Use as library

```python
import qldbtools as ql
```

## Command-line use

   Initial information collection requires a unique file path so it can be run
   repeatedly over DB collections with the same (owner,name) but other differences
   -- namely, in one or more of

   - creationTime
   - sha
   - cliVersion
   - language

   Those fields are collected and a single name addenum formed in
   `bin/mc-db-refine-info`. 

   The command sequence, grouped by data files, is

        cd ~/work-gh/mrva/mrvacommander/client/qldbtools
        ./bin/mc-db-initial-info ~/work-gh/mrva/mrva-open-source-download > db-info-1.csv
        ./bin/mc-db-refine-info < db-info-1.csv > db-info-2.csv
       
        ./bin/mc-db-view-info < db-info-2.csv &
        ./bin/mc-db-unique < db-info-2.csv > db-info-3.csv
        ./bin/mc-db-view-info < db-info-3.csv &

        ./bin/mc-db-populate-minio -n 23 < db-info-3.csv
        ./bin/mc-db-generate-selection -n 23 vscode-selection.json gh-mrva-selection.json < db-info-3.csv 
       
       
## Notes

   The preview-data plugin for VS Code has a bug; it displays `0` instead of
   `0e3379` for the following.  There are other entries with similar malfunction.
   
        CleverRaven,Cataclysm-DDA,0e3379,2.17.0,2024-05-08 12:13:10.038007+00:00,cpp,5ca7f4e59c2d7b0a93fb801a31138477f7b4a761,578098.0,/Users/hohn/work-gh/mrva/mrva-open-source-download/repos-2024-04-29/CleverRaven/Cataclysm-DDA/code-scanning/codeql/databases/cpp/db.zip,cpp,C/C++,1228.0,578098.0,2024-05-13T12:14:54.650648,cpp,True,4245,563435469
        CleverRaven,Cataclysm-DDA,3231f7,2.18.0,2024-07-18 11:13:01.673231+00:00,cpp,db3435138781937e9e0e999abbaa53f1d3afb5b7,579532.0,/Users/hohn/work-gh/mrva/mrva-open-source-download/repos/CleverRaven/Cataclysm-DDA/code-scanning/codeql/databases/cpp/db.zip,cpp,C/C++,1239.0,579532.0,2024-07-24T02:33:23.900885,cpp,True,1245,573213726
