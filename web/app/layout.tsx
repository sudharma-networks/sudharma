import type { Metadata } from "next";
import "./globals.css";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { SITE_URL } from "@/lib/project";

export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: { default: "Sudharma Network", template: "%s · Sudharma Network" },
  description: "Open-source Proof-of-Work blockchain infrastructure for users, miners, developers, researchers, and builders.",
  openGraph: { title: "Sudharma Network", description: "Open Blockchain. Open Development. Built for Everyone.", type: "website", url: SITE_URL }
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body><SiteHeader /><main>{children}</main><SiteFooter /></body></html>;
}
