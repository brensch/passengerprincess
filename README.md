# Passenger Princess Protector

A web application that helps protect passengers by providing safe routing and location-based services using OpenRouteService API.

## Prerequisites

- Node.js and npm (for build scripts)
- Firebase CLI installed and logged in
- OpenRouteService API key (get a free one from [openrouteservice.org](https://openrouteservice.org/))

## Setup

1. Clone this repository
2. Install dependencies:
   ```bash
   npm install
   ```
3. Copy the environment file and add your API key:
   ```bash
   cp .env.example .env
   ```
   Then edit `.env` and replace `your_openrouteservice_api_key_here` with your actual API key from [openrouteservice.org](https://openrouteservice.org/).

## API Key Security Note

Since this is a client-side application, the API key will be visible to users in the browser's developer tools. However, OpenRouteService API keys can be restricted by:

- IP address restrictions
- Referrer restrictions
- Daily/monthly usage limits

To avoid committing your API key to the repository, we use environment variables and a build process to inject the key into the HTML file. The `.env` file is included in `.gitignore` and should never be committed.

## Deployment

1. Make sure you've set up your `.env` file with your API key as described in the setup section.

2. Deploy to Firebase Hosting:
   ```bash
   firebase deploy
   ```

   The build process will automatically run before deployment, injecting your API key into the HTML file and copying it to the `public` directory.

## Development

To run locally:

1. Set your API key in the `.env` file
2. Build the project:
   ```bash
   npm run build
   ```
3. Open `public/index.html` in your browser

## Project Structure

- `index.html` - Source HTML file
- `public/index.html` - Built HTML file (generated during build)
- `firebase.json` - Firebase configuration
- `.env` - Environment variables (not committed to git)
- `build.js` - Build script to inject API key and copy to public

## License

[Add your license here]
