import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function GET() {
  const posts = store.getBlogPosts();
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
