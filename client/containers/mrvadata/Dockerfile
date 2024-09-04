# Use a minimal base image
FROM busybox

# This data container holds data from existing runs.  It cannot be built without
# ready data.


# Mirror parts of the host directory structure in the container
COPY tmp/dbstore-data /data/mrvacommander/dbstore-data

COPY tmp/scratch /data/mrvacommander/qldbtools/scratch

# Just run sh if this container is ever started
CMD ["sh"]
