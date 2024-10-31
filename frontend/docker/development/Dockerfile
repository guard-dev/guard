# Use the official Node.js image as the base
FROM node:18

# Set the working directory inside the container
WORKDIR /app

# Copy package.json, package-lock.json and other config files to the container
COPY . .

# Install dependencies
RUN rm -rf node_modules && npm ci --loglevel verbose

# Copy the app source code to the container
COPY . .

# Build the Next.js app
RUN npm run build --loglevel verbose
# RUN npm run build

# Expose the port the app will run on
EXPOSE 3000

# Start the app
CMD ["npm", "run", "dev"]
