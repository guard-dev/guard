import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";
import { ClerkProvider } from "@clerk/nextjs";
import { ThemeProvider } from "@/components/theme-provider";
import { ApolloProviderWrapper } from "./ApolloProviderWrapper";
import { CSPostHogProvider } from "./providers/PostHogProvider";

import { Analytics } from "@vercel/analytics/react"

const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-geist-sans",
  weight: "100 900",
});
const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
});

export const metadata: Metadata = {
  title: "Guard",
  description: "AI-Powered Cloud Security, Simplified.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const selfHosting = process.env.NEXT_PUBLIC_SELF_HOSTING === 'true';

  const content = (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <Analytics />
          {children}
        </ThemeProvider>
      </body>
    </html>
  );

  // Skip ClerkProvider and wrap directly with ApolloProvider for self-hosting
  if (selfHosting) {
    return <ApolloProviderWrapper>{content}</ApolloProviderWrapper>;
  }

  // Normal flow with Clerk authentication
  return (
    <ClerkProvider>
      <CSPostHogProvider>
        <ApolloProviderWrapper>{content}</ApolloProviderWrapper>
      </CSPostHogProvider>
    </ClerkProvider>
  );
}
