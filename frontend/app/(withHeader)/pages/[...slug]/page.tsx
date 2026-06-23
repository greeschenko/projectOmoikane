import { notFound } from "next/navigation";
import type { Metadata } from "next";
import store from "@/lib/store";
import { Container, Typography, Box, Breadcrumbs } from "@mui/material";
import Link from "next/link";

export const dynamic = "force-dynamic";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string[] }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const page = store.resolvePageByPath(slug);
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
  const page = store.resolvePageByPath(slug);
  if (!page || page.status !== "published") notFound();

  const breadcrumbs: { label: string; href: string }[] = [];
  let currentId: string | null = null;
  for (let i = 0; i < slug.length; i++) {
    const segment = slug[i];
    const p = store.getPageBySlug(segment, currentId);
    if (!p) break;
    currentId = p.id;
    breadcrumbs.push({
      label: p.title,
      href: "/pages/" + slug.slice(0, i + 1).join("/"),
    });
  }

  const parents = breadcrumbs.slice(0, -1);

  return (
    <Container maxWidth="md" sx={{ my: 4 }}>
      {parents.length > 0 && (
        <Breadcrumbs sx={{ mb: 1 }}>
          {parents.map((p, i) => (
            <Link
              key={i}
              href={p.href}
              style={{ textDecoration: "underline", color: "inherit" }}
            >
              {p.label}
            </Link>
          ))}
        </Breadcrumbs>
      )}
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
