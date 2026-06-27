import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const authorId = searchParams.get("authorId");
  let posts = store.getBlogPosts();
  if (authorId) {
    posts = posts.filter((p) => p.authorId === authorId);
  }
  return NextResponse.json(posts);
}

export async function POST(request: Request) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const body = await request.json();
  const post = store.createBlogPost({
    ...body,
    authorId: session.userId,
  });
  return NextResponse.json(post, { status: 201 });
}
