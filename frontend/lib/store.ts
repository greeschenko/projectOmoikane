import crypto from "node:crypto";

export interface User {
  id: string;
  name: string;
  email: string;
  password: string;
  role: "admin" | "user";
  createdAt: string;
  updatedAt: string;
}

export interface Page {
  id: string;
  title: string;
  slug: string;
  content: string;
  metaTitle?: string;
  metaDescription?: string;
  metaKeywords?: string;
  parentId: string | null;
  sortOrder: number;
  published: boolean;
  createdAt: string;
  updatedAt: string;
}

function hashPassword(password: string): string {
  const salt = crypto.randomBytes(16).toString("hex");
  const hash = crypto.pbkdf2Sync(password, salt, 1000, 64, "sha512").toString("hex");
  return `${salt}:${hash}`;
}

export function verifyPassword(password: string, stored: string): boolean {
  const [salt, hash] = stored.split(":");
  const check = crypto.pbkdf2Sync(password, salt, 1000, 64, "sha512").toString("hex");
  return hash === check;
}

function stripPassword(user: User): Omit<User, "password"> {
  const { password, ...rest } = user;
  return rest;
}

class InMemoryStore {
  private users = new Map<string, User>();
  private pages = new Map<string, Page>();

  getUsers(): Omit<User, "password">[] {
    return Array.from(this.users.values()).map(stripPassword);
  }

  getUser(id: string): User | undefined {
    return this.users.get(id);
  }

  getUserByEmail(email: string): User | undefined {
    for (const user of this.users.values()) {
      if (user.email === email) return user;
    }
    return undefined;
  }

  createUser(data: {
    name: string;
    email: string;
    password: string;
    role: "admin" | "user";
  }): Omit<User, "password"> {
    const id = crypto.randomUUID();
    const now = new Date().toISOString();
    const user: User = {
      id,
      name: data.name,
      email: data.email,
      password: hashPassword(data.password),
      role: data.role,
      createdAt: now,
      updatedAt: now,
    };
    this.users.set(id, user);
    return stripPassword(user);
  }

  updateUser(
    id: string,
    data: Partial<Omit<User, "id" | "createdAt">>
  ): Omit<User, "password"> | undefined {
    const user = this.users.get(id);
    if (!user) return undefined;
    const updated: User = {
      ...user,
      ...data,
      password: data.password ? hashPassword(data.password) : user.password,
      updatedAt: new Date().toISOString(),
    };
    this.users.set(id, updated);
    return stripPassword(updated);
  }

  deleteUser(id: string): boolean {
    return this.users.delete(id);
  }

  get userCount(): number {
    return this.users.size;
  }

  getPages(): Page[] {
    return Array.from(this.pages.values());
  }

  getPage(id: string): Page | undefined {
    return this.pages.get(id);
  }

  getPageBySlug(slug: string, parentId: string | null = null): Page | undefined {
    for (const page of this.pages.values()) {
      if (page.slug === slug && page.parentId === parentId) return page;
    }
    return undefined;
  }

  createPage(data: {
    title: string;
    slug: string;
    content: string;
    metaTitle?: string;
    metaDescription?: string;
    metaKeywords?: string;
    parentId?: string | null;
    published?: boolean;
  }): Page {
    const id = crypto.randomUUID();
    const now = new Date().toISOString();
    const page: Page = {
      id,
      title: data.title,
      slug: data.slug,
      content: data.content,
      metaTitle: data.metaTitle,
      metaDescription: data.metaDescription,
      metaKeywords: data.metaKeywords,
      parentId: data.parentId ?? null,
      sortOrder: 0,
      published: data.published ?? false,
      createdAt: now,
      updatedAt: now,
    };
    this.pages.set(id, page);
    return page;
  }

  updatePage(
    id: string,
    data: Partial<Omit<Page, "id" | "createdAt">>
  ): Page | undefined {
    const page = this.pages.get(id);
    if (!page) return undefined;
    const updated: Page = {
      ...page,
      ...data,
      updatedAt: new Date().toISOString(),
    };
    this.pages.set(id, updated);
    return updated;
  }

  deletePage(id: string): boolean {
    const children = Array.from(this.pages.values()).filter((p) => p.parentId === id);
    for (const child of children) {
      this.deletePage(child.id);
    }
    return this.pages.delete(id);
  }

  resolvePageByPath(segments: string[]): Page | undefined {
    let parentId: string | null = null;
    for (const segment of segments) {
      const page = this.getPageBySlug(segment, parentId);
      if (!page) return undefined;
      parentId = page.id;
    }
    return parentId ? this.getPage(parentId!) : undefined;
  }
}

const GLOBAL_KEY = "__omoikane_store__";

function getGlobalStore(): InMemoryStore {
  if (typeof globalThis !== "undefined") {
    const existing = (globalThis as any)[GLOBAL_KEY];
    if (existing) return existing;
    const store = new InMemoryStore();
    (globalThis as any)[GLOBAL_KEY] = store;
    return store;
  }
  // Fallback for environments without globalThis (edge)
  const store = new InMemoryStore();
  return store;
}

const store = getGlobalStore();
export default store;
