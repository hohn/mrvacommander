const crypto = require("crypto");

const atl_L = [[1, "s1"], [2, "str"]];
const sl_hash = atl_L.map(item => {
    const json = JSON.stringify(item);
    return crypto.createHash("md5").update(json).digest("hex");
});

console.log(sl_hash);
