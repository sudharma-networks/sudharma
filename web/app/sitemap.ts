import type { MetadataRoute } from "next";
import { PRIMARY_NAV } from "@/lib/navigation";
import { SITE_URL } from "@/lib/project";
const mining = ["nvidia", "amd", "solo", "pools", "kryptex", "benchmarks", "troubleshooting"].map((x) => `/mining/${x}`);
const dev = ["getting-started", "node", "rpc", "wallets", "payments", "contributing", "protocol"].map((x) => `/developers/${x}`);
export default function sitemap(): MetadataRoute.Sitemap { return [...PRIMARY_NAV.map(([, href]) => href), ...mining, ...dev].map((path) => ({ url: `${SITE_URL}${path === "/" ? "" : path}`, changeFrequency: "weekly" as const, priority: path === "/" ? 1 : 0.7 })); }
