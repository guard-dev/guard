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
  return (
    <ClerkProvider>
      <ApolloProviderWrapper>
        <html lang="en">
          <CSPostHogProvider>
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
          </CSPostHogProvider>
        </html>
      </ApolloProviderWrapper>
    </ClerkProvider>
  );
}
