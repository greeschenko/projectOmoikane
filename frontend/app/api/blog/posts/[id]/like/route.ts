import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function POST(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const { id } = await params;
  const result = store.toggleLike(id, session.userId);
  return NextResponse.json(result);
}
