import { getSession } from "@/lib/auth";
import PublicHeader from "@/components/PublicHeader";
import PublicFooter from "@/components/PublicFooter";
import { Box } from "@mui/material";

const jsonLd = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  name: "Omoikane",
  description: "Project Omoikane",
};

export default async function PublicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await getSession();
  return (
    <Box sx={{ display: "flex", flexDirection: "column", minHeight: "100vh" }}>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <PublicHeader session={session} />
      <Box component="main" id="main-content" sx={{ flexGrow: 1 }}>
        {children}
      </Box>
      <PublicFooter />
    </Box>
  );
}
