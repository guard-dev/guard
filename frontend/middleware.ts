import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

// Default middleware for self-hosted mode
function selfHostedMiddleware(request: NextRequest) {
  return NextResponse.next();
}

// Create the middleware based on hosting mode
const middleware = () => {
  if (process.env.NEXT_PUBLIC_SELF_HOSTING === 'true') {
    return selfHostedMiddleware;
  }

  // Only import and use Clerk in non-self-hosted mode
  const { clerkMiddleware, createRouteMatcher } = require("@clerk/nextjs/server");
  const isPublicRoute = createRouteMatcher(['/sign-in(.*)', '/sign-up(.*)', '/'])

  return clerkMiddleware((auth: any, request: any) => {
    if (!isPublicRoute(request)) {
      auth().protect()
    }
  });
}

export default middleware();

export const config = {
  matcher: [
    // Skip Next.js internals and all static files, unless found in search params
    '/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)',
    // Always run for API routes
    '/(api|trpc)(.*)',
  ],
};
