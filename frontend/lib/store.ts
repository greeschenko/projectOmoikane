import crypto from "node:crypto";

export interface User {
  id: string;
  name: string;
  email: string;
  password: string;
  role: "admin" | "user";
  status: "active" | "banned";
  avatar?: string;
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
  status: "draft" | "published";
  inMenu: boolean;
  previewToken: string;
  createdAt: string;
  updatedAt: string;
}

export interface Message {
  id: string;
  title: string;
  content: string;
  createdAt: string;
  readBy: string[];
}

export interface SiteSettings {
  siteName: string;
  tagline: string;
  logo: string;
  favicon: string;
}

export interface MediaItem {
  id: string;
  filename: string;
  mimeType: string;
  size: number;
  data: string;
  createdAt: string;
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
  private messages = new Map<string, Message>();
  private medias = new Map<string, MediaItem>();
  private settings: SiteSettings = {
    siteName: "Omoikane",
    tagline: "A modern CMS",
    logo: "",
    favicon: "",
  };

  // --- Users ---

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
    status?: "active" | "banned";
  }): Omit<User, "password"> {
    const id = crypto.randomUUID();
    const now = new Date().toISOString();
    const user: User = {
      id,
      name: data.name,
      email: data.email,
      password: hashPassword(data.password),
      role: data.role,
      status: data.status ?? "active",
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

  // --- Pages ---

  getPages(): Page[] {
    return Array.from(this.pages.values());
  }

  getMenuPages(): Page[] {
    return Array.from(this.pages.values()).filter(
      (p) => p.inMenu && p.status === "published"
    );
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
    status?: "draft" | "published";
    inMenu?: boolean;
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
      status: data.status ?? (data.published ? "published" : "draft"),
      inMenu: data.inMenu ?? false,
      previewToken: crypto.randomUUID(),
      createdAt: now,
      updatedAt: now,
    };
    this.pages.set(id, page);
    return page;
  }

  updatePage(
    id: string,
    data: Partial<Omit<Page, "id" | "createdAt"> & { published?: boolean }>
  ): Page | undefined {
    const page = this.pages.get(id);
    if (!page) return undefined;
    const updated: Page = {
      ...page,
      ...data,
      status: data.status ?? (data.published !== undefined ? (data.published ? "published" : "draft") : page.status),
      inMenu: data.inMenu ?? page.inMenu,
      previewToken: page.previewToken,
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

  reorderPages(parentId: string | null, pageIds: string[]): void {
    for (let i = 0; i < pageIds.length; i++) {
      const page = this.pages.get(pageIds[i]);
      if (page && page.parentId === parentId) {
        page.sortOrder = i;
        page.updatedAt = new Date().toISOString();
      }
    }
  }

  // --- Messages ---

  getMessages(): Message[] {
    return Array.from(this.messages.values()).sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }

  createMessage(data: { title: string; content: string }): Message {
    const id = crypto.randomUUID();
    const now = new Date().toISOString();
    const message: Message = { id, title: data.title, content: data.content, createdAt: now, readBy: [] };
    this.messages.set(id, message);
    return message;
  }

  markAsRead(messageId: string, userId: string): Message | undefined {
    const msg = this.messages.get(messageId);
    if (!msg) return undefined;
    if (!msg.readBy.includes(userId)) {
      msg.readBy.push(userId);
    }
    return msg;
  }

  getUnreadCount(userId: string): number {
    let count = 0;
    for (const msg of this.messages.values()) {
      if (!msg.readBy.includes(userId)) count++;
    }
    return count;
  }

  clearMessages(): void {
    this.messages.clear();
  }

  markAllAsRead(userId: string): void {
    for (const msg of this.messages.values()) {
      if (!msg.readBy.includes(userId)) {
        msg.readBy.push(userId);
      }
    }
  }

  // --- Media ---

  getMedia(): MediaItem[] {
    return Array.from(this.medias.values()).sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    );
  }

  createMedia(data: {
    filename: string;
    mimeType: string;
    size: number;
    data: string;
  }): MediaItem {
    const id = crypto.randomUUID();
    const now = new Date().toISOString();
    const item: MediaItem = { id, ...data, createdAt: now };
    this.medias.set(id, item);
    return item;
  }

  deleteMedia(id: string): boolean {
    return this.medias.delete(id);
  }

  // --- Site Settings ---

  getSettings(): SiteSettings {
    return { ...this.settings };
  }

  updateSettings(data: Partial<SiteSettings>): SiteSettings {
    this.settings = { ...this.settings, ...data };
    return { ...this.settings };
  }

  // --- Dashboard Stats ---

  getDashboardStats() {
    const now = new Date();
    const sevenDaysAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
    const usersArray = Array.from(this.users.values());
    const recentUsers = usersArray.filter((u) => new Date(u.createdAt) >= sevenDaysAgo);
    const registrationsByDay: Record<string, number> = {};
    for (let i = 0; i < 7; i++) {
      const d = new Date(now.getTime() - i * 24 * 60 * 60 * 1000);
      const key = d.toISOString().slice(0, 10);
      registrationsByDay[key] = 0;
    }
    for (const u of recentUsers) {
      const key = u.createdAt.slice(0, 10);
      if (registrationsByDay[key] !== undefined) {
        registrationsByDay[key]++;
      }
    }
    return {
      userCount: this.users.size,
      pageCount: this.pages.size,
      recentRegistrations: Object.entries(registrationsByDay)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([date, count]) => ({ date, count })),
    };
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
  const store = new InMemoryStore();
  return store;
}

const store = getGlobalStore();
export default store;
