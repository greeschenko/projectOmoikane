import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function GET() {
  const categories = store.getCategories();
  return NextResponse.json(categories);
}

export async function POST(request: Request) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const body = await request.json();
  const category = store.createCategory(body.name, body.slug, body.description);
  return NextResponse.json(category, { status: 201 });
}
