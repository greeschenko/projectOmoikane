import { getSession } from "@/lib/auth";
import PublicHeader from "@/components/PublicHeader";
import PublicFooter from "@/components/PublicFooter";
import { Box } from "@mui/material";

export default async function PublicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await getSession();
  return (
    <Box sx={{ display: "flex", flexDirection: "column", minHeight: "100vh" }}>
      <PublicHeader session={session} />
      <Box component="main" sx={{ flexGrow: 1 }}>
        {children}
      </Box>
      <PublicFooter />
    </Box>
  );
}
