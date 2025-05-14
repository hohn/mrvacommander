import hashlib
import json

atl_L = [(1, "s1"), (2, "str")]
sl_hash = []

for item in atl_L:
    encoded = json.dumps(item, separators=(',', ':')).encode("utf-8")
    md5sum = hashlib.md5(encoded).hexdigest()
    sl_hash.append(md5sum)

print(sl_hash)
