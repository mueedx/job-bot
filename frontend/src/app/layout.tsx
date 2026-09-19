import type { Metadata } from "next";
import { AppNav } from "@/components/AppNav";
import "@fontsource/ibm-plex-sans/400.css";
import "@fontsource/ibm-plex-sans/500.css";
import "@fontsource/ibm-plex-sans/600.css";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";
import "./globals.css";

export const metadata: Metadata = {
  title: "Job Agent Control Center",
  description: "Localhost pipeline board for Stage 1 job application calibration",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen antialiased">
        <AppNav />
        <main className="mx-auto max-w-[1400px] px-4 pb-16 pt-6 md:px-6">{children}</main>
      </body>
    </html>
  );
}
