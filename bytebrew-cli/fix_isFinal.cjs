const fs = require('fs');

const file = process.argv[2];
const content = fs.readFileSync(file, 'utf8');

// Add isFinal: false to simulateResponse calls without it
const fixed = content.replace(
  /gateway\.simulateResponse\(\{([^}]*?)\}\);/gs,
  (match, props) => {
    // Skip if already has isFinal
    if (props.includes('isFinal')) {
      return match;
    }
    // Add isFinal: false before closing brace
    const trimmedProps = props.trimEnd();
    return `gateway.simulateResponse({${trimmedProps},\n        isFinal: false,\n      });`;
  }
);

fs.writeFileSync(file, fixed, 'utf8');
