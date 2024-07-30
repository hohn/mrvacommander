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

   XX: Add `mc-db-generate-selection`

   The command sequence, grouped by data files, is

        cd ~/work-gh/mrva/mrvacommander/client/qldbtools
        ./bin/mc-db-initial-info ~/work-gh/mrva/mrva-open-source-download > db-info-1.csv
        ./bin/mc-db-refine-info < db-info-1.csv > db-info-2.csv
       
        ./bin/mc-db-view-info < db-info-2.csv &
        ./bin/mc-db-unique < db-info-2.csv > db-info-3.csv
        ./bin/mc-db-view-info < db-info-3.csv &

        ./bin/mc-db-populate-minio -n 23 < db-info-3.csv
        ./bin/mc-db-generate-selection -n 23 vscode-selection.json gh-mrva-selection.json < db-info-3.csv 
       
       
               
