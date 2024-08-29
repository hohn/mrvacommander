# Use a minimal base image
FROM busybox

# This data container holds data from existing runs.  It cannot be built without
# ready data.

# Before running docker build, make sure you have set up the directory mirrors
# dbstore-data and scratch.

# E.g.:
    # cd ~/work-gh/mrva/mrvacommander/client/containers/mrvadata/
    # mkdir tmp
    # cp -r ../../../dbstore-data   tmp
    # cp -r ../../qldbtools/scratch tmp

# Mirror parts of the host directory structure in the container
COPY tmp/dbstore-data /data/mrvacommander/dbstore-data

COPY tmp/scratch /data/mrvacommander/qldbtools/scratch

# Just run sh if this container is ever started
CMD ["sh"]
