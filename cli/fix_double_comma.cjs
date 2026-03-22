const fs = require('fs');
const file = process.argv[2];
const content = fs.readFileSync(file, 'utf8');
const fixed = content.replace(/,,/g, ',');
fs.writeFileSync(file, fixed, 'utf8');
