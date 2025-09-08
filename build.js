const fs = require('fs');
const path = require('path');

// Load environment variables from .env file
require('dotenv').config();

const apiKey = process.env.ORS_API_KEY;
if (!apiKey) {
  console.error('Error: ORS_API_KEY not found in .env file');
  process.exit(1);
}

// Read the source HTML file from root
const sourceFilePath = path.join(__dirname, 'index.html');
let htmlContent = fs.readFileSync(sourceFilePath, 'utf8');

// Replace the placeholder with the actual API key
htmlContent = htmlContent.replace(/YOUR_OPENROUTESERVICE_API_KEY/g, apiKey);

// Ensure public directory exists
const publicDir = path.join(__dirname, 'public');
if (!fs.existsSync(publicDir)) {
  fs.mkdirSync(publicDir);
}

// Write the updated HTML to public/index.html
const outputFilePath = path.join(publicDir, 'index.html');
fs.writeFileSync(outputFilePath, htmlContent);

console.log('Build complete: index.html built into public/index.html with API key injected');
