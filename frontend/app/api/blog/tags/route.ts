import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function GET() {
  const tags = store.getTags();
  return NextResponse.json(tags);
}

export async function POST(request: Request) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const body = await request.json();
  const tag = store.createTag(body.name, body.slug);
  return NextResponse.json(tag, { status: 201 });
}
