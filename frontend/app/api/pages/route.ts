import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAdmin } from "@/lib/auth";

export async function GET(request: Request) {
  const url = new URL(request.url);
  if (url.searchParams.get("menu") === "true") {
    return NextResponse.json(store.getMenuPages());
  }
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
  const { title, slug, content, metaTitle, metaDescription, metaKeywords, parentId, published, status, inMenu } = body;
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
    status,
    inMenu,
  });
  return NextResponse.json(page, { status: 201 });
}
