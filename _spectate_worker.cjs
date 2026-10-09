const fs = require('fs');
const path = require('path');
const dir = 'Public';
const html = fs.readFileSync(path.join(dir, 'index.html'), 'utf8');
const wins = [...html.matchAll(/window\.([A-Za-z0-9_]+)\s*(?==|\()/g)].map(m => m[1]);
const ids = [...html.matchAll(/id="([a-zA-Z0-9_-]+)"/g)].map(m => m[1]);
const srcs = [...html.matchAll(/<script[^>]+src="([^"]+)"/g)].map(m => m[1]);
const inline = [...html.matchAll(/<script>([^<]{5,})<\/script>/g)].map(m => m[1].trim());
const out = [
  '=== window.* HANDLERS ('+wins.length+') ===',
  ...wins.map((w,i)=>('  '+i+': window.'+w)),
  '=== TOTAL window handlers ===', wins.length,
  '=== script srcs ===',
  srcs.map(s=>'  '+s).join('\n'),
  '=== inline script chunks ('+inline.length+') ===',
  ...inline.map((s,i)=>('  '+i+': '+s.slice(0,80)+'...')),
].join('\n');
fs.writeFileSync('_spectate_api.txt', out);
