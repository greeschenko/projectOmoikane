import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { apiFetch } from "@/lib/api";
import { Container, Typography, Box, Chip } from "@mui/material";

export const dynamic = "force-dynamic";

export async function generateMetadata({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ token?: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const { token } = await searchParams;
  try {
    const page = await apiFetch<Record<string, unknown>>(`/pages/${id}`);
    if (page.previewToken !== token) return { title: "Not Found" };
    return {
      title: `${page.title} (Preview)`,
    };
  } catch {
    return { title: "Not Found" };
  }
}

export default async function PreviewPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ token?: string }>;
}) {
  const { id } = await params;
  const { token } = await searchParams;
  try {
    const page = await apiFetch<Record<string, unknown>>(`/pages/${id}`);
    if (page.previewToken !== token) notFound();

    return (
      <Container maxWidth="md" sx={{ my: 4 }}>
        <Box sx={{ mb: 2, display: "flex", alignItems: "center", gap: 2 }}>
          <Typography variant="h4" component="h1" gutterBottom sx={{ mb: 0 }}>
            {page.title as string}
          </Typography>
          <Chip label={page.status === "draft" ? "Draft" : "Published"} size="small" color={page.status === "draft" ? "default" : "success"} />
        </Box>
        <Box
          sx={{
            borderTop: 1, borderColor: "divider", pt: 2,
            "& img": { maxWidth: "100%", height: "auto" },
            "& p": { mb: 1 },
          }}
          dangerouslySetInnerHTML={{ __html: page.content as string }}
        />
      </Container>
    );
  } catch {
    notFound();
  }
}
