import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAdmin } from "@/lib/auth";

export async function GET() {
  return NextResponse.json(store.getPages());
}

export async function POST(request: Request) {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const body = await request.json();
  const { title, slug, content, metaTitle, metaDescription, metaKeywords, parentId, published } = body;
  if (!title || !slug || !content) {
    return NextResponse.json({ error: "Title, slug, and content required" }, { status: 400 });
  }
  const page = store.createPage({
    title,
    slug,
    content,
    metaTitle,
    metaDescription,
    metaKeywords,
    parentId,
    published,
  });
  return NextResponse.json(page, { status: 201 });
}
