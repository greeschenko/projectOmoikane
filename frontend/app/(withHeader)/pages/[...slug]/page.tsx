import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { apiFetch } from "@/lib/api";
import { Container, Typography, Box, Breadcrumbs } from "@mui/material";
import Link from "next/link";

export const dynamic = "force-dynamic";

interface PageData {
  id: string;
  title: string;
  slug: string;
  content: string;
  status: string;
  metaTitle?: string;
  metaDescription?: string;
  parentId?: string | null;
  parentTitle?: string;
  parentSlug?: string;
}

async function fetchPageBySlug(slug: string): Promise<PageData | null> {
  try {
    return await apiFetch<PageData>(`/pages/slug/${slug}`);
  } catch {
    return null;
  }
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string[] }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const page = await fetchPageBySlug(slug[slug.length - 1]);
  if (!page) return { title: "Not Found" };
  return {
    title: page.metaTitle || page.title,
    description: page.metaDescription,
  };
}

export default async function PublicPage({
  params,
}: {
  params: Promise<{ slug: string[] }>;
}) {
  const { slug } = await params;
  const page = await fetchPageBySlug(slug[slug.length - 1]);
  if (!page || page.status !== "published") notFound();

  return (
    <Container maxWidth="md" sx={{ my: 4 }}>
      <Breadcrumbs aria-label="breadcrumb" sx={{ mb: 2 }}>
        <Link href="/" style={{ textDecoration: "none", color: "inherit" }}>
          Home
        </Link>
        {page.parentSlug && page.parentTitle && (
          <Link
            href={`/pages/${page.parentSlug}`}
            style={{ textDecoration: "none", color: "inherit" }}
          >
            {page.parentTitle}
          </Link>
        )}
        <Typography color="text.primary">{page.title}</Typography>
      </Breadcrumbs>
      <Typography variant="h4" component="h1" gutterBottom>
        {page.title}
      </Typography>
      <Box
        sx={{
          "& img": { maxWidth: "100%", height: "auto" },
          "& p": { mb: 1 },
        }}
        dangerouslySetInnerHTML={{ __html: page.content }}
      />
    </Container>
  );
}
