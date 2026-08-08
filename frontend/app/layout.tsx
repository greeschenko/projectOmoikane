import type { Metadata } from "next";
import ThemeRegistry from "@/components/ThemeRegistry";
import "./globals.css";

export const metadata: Metadata = {
  title: "Omoikane",
  description: "Project Omoikane",
  openGraph: {
    title: "Omoikane",
    description: "Project Omoikane",
    type: "website",
  },
  twitter: {
    card: "summary",
    title: "Omoikane",
    description: "Project Omoikane",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <a href="#main-content" className="skip-link">
          Skip to content
        </a>
        <ThemeRegistry>{children}</ThemeRegistry>
      </body>
    </html>
  );
}
