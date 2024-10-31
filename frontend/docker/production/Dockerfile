# Use the official Node.js image as the base
FROM node:18

# Set the working directory inside the container
WORKDIR /app

# Copy package.json, package-lock.json and other config files to the container
COPY . .

ARG NEXT_PUBLIC_API_BASE_URL
ARG NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY

ARG NEXT_SERVER_API_BASE_URL
ARG NEXT_PUBLIC_POSTHOG_KEY
ARG NEXT_PUBLIC_POSTHOG_HOST
ARG NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY

# Install dependencies
RUN npm ci --loglevel verbose

# Copy the app source code to the container
COPY . .

ENV NODE_ENV=production

ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY=$NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY
ENV NEXT_SERVER_API_BASE_URL=$NEXT_SERVER_API_BASE_URL
ENV NEXT_PUBLIC_POSTHOG_KEY=$NEXT_PUBLIC_POSTHOG_KEY
ENV NEXT_PUBLIC_POSTHOG_HOST=$NEXT_PUBLIC_POSTHOG_HOST
ENV NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=$NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY

# Build the Next.js app
RUN npm run build --loglevel verbose

# Expose the port the app will run on
EXPOSE 3000

# Start the app
CMD ["npm", "start"]
