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

    cd ~/work-gh/mrva/mrvacommander/client/qldbtools
    ./bin/mc-db-initial-info ~/work-gh/mrva/mrva-open-source-download > db-info-1.csv
    
    ./bin/mc-db-refine-info < db-info-1.csv > db-info-2.csv
    
    ./bin/mc-db-populate-minio < db-info-2.csv -n 3
    
        
