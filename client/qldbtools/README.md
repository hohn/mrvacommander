# Introduction to qldbtools

`qldbtools` is a Python package for selecting sets of CodeQL databases to work on.
It uses a (pandas) dataframe in the implementation, but all results sets are
available as CSV files to provide flexibility in the tools you want to work with.

The rationale is simple: When working with larger collections of CodeQL databases,
spread over time, languages, etc., many criteria can be used to select the subset
of interest.  This package addresses that aspect of MRVA (multi repository
variant analysis). 

For example, consider this scenario from an enterprise.  We have 10,000
repositories in C/C++, 5,000 in Python.  We build CodeQL dabases weekly and keep
the last 2 years worth.
This means for the last 2 years there are

    (10000 + 5000) * 52 * 2 = 1560000

databases to select from for a single MRVA run.  1.5 Million rows are readily
handled by a pandas (or R) dataframe.  

The full list of criteria currently encoded via the columns is

-   owner
-   name
-   CID
-   cliVersion
-   creationTime
-   language
-   sha -- git commit sha of the code the CodeQL database is built against
-   baselineLinesOfCode
-   path
-   db_lang
-   db_lang_displayName
-   db_lang_file_count
-   db_lang_linesOfCode
-   ctime
-   primaryLanguage
-   finalised
-   left_index
-   size

The minimal criteria needed to distinguish databases in the above scenario are

-   cliVersion
-   creationTime
-   language
-   sha

These are encoded in the single custom id column 'CID'.

Thus, a database can be fully specified using a (owner,name,CID) tuple and this is
encoded in the names used by the MRVA server and clients. The selection of
databases can of course be done using the whole table.

For an example of the workflow, see [section 'command line use'](#command-line-use).



A small sample of a full table:

|    | owner    | name           | CID    | cliVersion   | creationTime                     | language   | sha                                      |   baselineLinesOfCode | path                                                                                                                          | db_lang     | db_lang_displayName   |   db_lang_file_count |   db_lang_linesOfCode | ctime                      | primaryLanguage   |   finalised |   left_index |     size |
|---:|:---------|:---------------|:-------|:-------------|:---------------------------------|:-----------|:-----------------------------------------|----------------------:|:------------------------------------------------------------------------------------------------------------------------------|:------------|:----------------------|---------------------:|----------------------:|:---------------------------|:------------------|------------:|-------------:|---------:|
|  0 | 1adrianb | face-alignment | 1f8d99 | 2.16.1       | 2024-02-08 14:18:20.983830+00:00 | python     | c94dd024b1f5410ef160ff82a8423141e2bbb6b4 |                  1839 | /Users/hohn/work-gh/mrva/mrva-open-source-download/repos/1adrianb/face-alignment/code-scanning/codeql/databases/python/db.zip | python      | Python                |                   25 |                  1839 | 2024-07-24T14:09:02.187201 | python            |           1 |         1454 | 24075001 |
|  1 | 2shou    | TextGrocery    | 9ab87a | 2.12.1       | 2023-02-17T11:32:30.863093193Z   | cpp        | 8a4e41349a9b0175d9a73bc32a6b2eb6bfb51430 |                  3939 | /Users/hohn/work-gh/mrva/mrva-open-source-download/repos/2shou/TextGrocery/code-scanning/codeql/databases/cpp/db.zip          | no-language | no-language           |                    0 |                    -1 | 2024-07-24T06:25:55.347568 | cpp               |         nan |         1403 |  3612535 |
|  2 | 3b1b     | manim          | 76fdc7 | 2.17.5       | 2024-06-27 17:37:20.587627+00:00 | python     | 88c7e9d2c96be1ea729b089c06cabb1bd3b2c187 |                 19905 | /Users/hohn/work-gh/mrva/mrva-open-source-download/repos/3b1b/manim/code-scanning/codeql/databases/python/db.zip              | python      | Python                |                   94 |                 19905 | 2024-07-24T13:23:04.716286 | python            |           1 |         1647 | 26407541 |

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
   The best way to examine the code is starting from the high-level scripts in
   `bin/`. 

## Command line use

   Initial information collection requires a unique file path so it can be run
   repeatedly over DB collections with the same (owner,name) but other differences
   -- namely, in one or more of

   - creationTime
   - sha
   - cliVersion
   - language

   Those fields are collected in `bin/mc-db-refine-info`. 

   An example workflow with commands grouped by data files follows.

        cd ~/work-gh/mrva/mrvacommander/client/qldbtools && mkdir -p scratch
        ./bin/mc-db-initial-info ~/work-gh/mrva/mrva-open-source-download > scratch/db-info-1.csv
        ./bin/mc-db-refine-info < scratch/db-info-1.csv > scratch/db-info-2.csv
       
        ./bin/mc-db-view-info < scratch/db-info-2.csv &
        ./bin/mc-db-unique cpp < scratch/db-info-2.csv > scratch/db-info-3.csv
        ./bin/mc-db-view-info < scratch/db-info-3.csv &

        ./bin/mc-db-populate-minio -n 11 < scratch/db-info-3.csv
        ./bin/mc-db-generate-selection -n 11 \
            scratch/vscode-selection.json \
            scratch/gh-mrva-selection.json \
            < scratch/db-info-3.csv 


   To see the full information for a selection, use `mc-rows-from-mrva-list`:
   
        ./bin/mc-rows-from-mrva-list scratch/gh-mrva-selection.json \
            scratch/db-info-3.csv > scratch/selection-full-info

   To check, e.g., the `language` column:

        csvcut -c language scratch/selection-full-info 

## Notes

   The `preview-data` plugin for VS Code has a bug; it displays `0` instead of
   `0e3379` for the following.  There are other entries with similar malfunction.
   
        CleverRaven,Cataclysm-DDA,0e3379,2.17.0,2024-05-08 12:13:10.038007+00:00,cpp,5ca7f4e59c2d7b0a93fb801a31138477f7b4a761,578098.0,/Users/hohn/work-gh/mrva/mrva-open-source-download/repos-2024-04-29/CleverRaven/Cataclysm-DDA/code-scanning/codeql/databases/cpp/db.zip,cpp,C/C++,1228.0,578098.0,2024-05-13T12:14:54.650648,cpp,True,4245,563435469
        CleverRaven,Cataclysm-DDA,3231f7,2.18.0,2024-07-18 11:13:01.673231+00:00,cpp,db3435138781937e9e0e999abbaa53f1d3afb5b7,579532.0,/Users/hohn/work-gh/mrva/mrva-open-source-download/repos/CleverRaven/Cataclysm-DDA/code-scanning/codeql/databases/cpp/db.zip,cpp,C/C++,1239.0,579532.0,2024-07-24T02:33:23.900885,cpp,True,1245,573213726
