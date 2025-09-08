const fs = require('fs');
const path = require('path');

// Load environment variables from .env file
require('dotenv').config();

const apiKey = process.env.ORS_API_KEY;
if (!apiKey) {
  console.error('Error: ORS_API_KEY not found in .env file');
  process.exit(1);
}

console.log('API Key from env:', apiKey);

// Read the source HTML file from root
const sourceFilePath = path.join(__dirname, 'index.html');
let htmlContent = fs.readFileSync(sourceFilePath, 'utf8');

// Replace the placeholder with the actual API key
// Only replace the assignment line, not the check
htmlContent = htmlContent.replace(
  /const ORS_API_KEY = 'YOUR_OPENROUTESERVICE_API_KEY'/g,
  `const ORS_API_KEY = '${apiKey}'`
);

console.log('After replacement:', htmlContent.substring(htmlContent.indexOf('const ORS_API_KEY') - 10, htmlContent.indexOf('const ORS_API_KEY') + 50));

// Ensure public directory exists
const publicDir = path.join(__dirname, 'public');
if (!fs.existsSync(publicDir)) {
  fs.mkdirSync(publicDir);
}

// Write the updated HTML to public/index.html
const outputFilePath = path.join(publicDir, 'index.html');
fs.writeFileSync(outputFilePath, htmlContent);

console.log('Build complete: index.html built into public/index.html with API key injected');
