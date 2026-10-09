const fs = require('fs');
const html = fs.readFileSync('Public/index.html', 'utf8');
const divs = html.split('\n').map((l, i) => `${i}: ${l}`).join('\n');
fs.writeFileSync('_index_btns.txt', divs);
