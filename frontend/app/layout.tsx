import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "MEV Arbitrage Dashboard",
  description: "Trading Terminal for Go-lang Bot",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body>
        {children}
      </body>
    </html>
  );
}
