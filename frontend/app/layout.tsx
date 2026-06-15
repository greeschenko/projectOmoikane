import type { Metadata } from "next";
import ThemeRegistry from "@/components/ThemeRegistry";
import "./globals.css";

export const metadata: Metadata = {
  title: "Omoikane",
  description: "Project Omoikane",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <ThemeRegistry>{children}</ThemeRegistry>
      </body>
    </html>
  );
}
