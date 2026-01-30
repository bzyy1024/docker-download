import "./globals.css";
import type { Metadata } from "next";
import { ThemeProvider } from "@/components/theme-provider";
import { TopBar } from "@/components/top-bar";

export const metadata: Metadata = {
  title: "Docker Image Downloader - Fast & Reliable",
  description:
    "Effortlessly download Docker images from custom registries, optimized for restricted networks. Supports authentication, deduplication, and more.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh" suppressHydrationWarning>
      <body className="min-h-screen bg-background font-sans antialiased">
        <ThemeProvider>
          <div className="relative flex min-h-screen flex-col">
            <TopBar />
            <main className="flex-1">
              <div className="mx-auto w-full max-w-5xl px-4 py-8 sm:px-6 lg:px-10">
                {children}
              </div>
            </main>
          </div>
        </ThemeProvider>
      </body>
    </html>
  );
}
