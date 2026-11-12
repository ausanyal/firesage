import type { PlanInput, SavedPlanRecord } from '../types';

const DB_NAME = 'firesage_local_db';
const DB_VERSION = 1;
const STORE_PLANS = 'saved_plans';
const ACTIVE_DRAFT_KEY = 'firesage_active_draft_v1';

/**
 * ClientPlanRepository defines the storage & sync seam.
 * Default V1 implementation is 100% Local-First via IndexedDB + JSON Export/Import,
 * designed to accept a CloudFirestoreSyncAdapter (Google Sign-In) with zero UI refactoring.
 */
export interface ClientPlanRepository {
  listPlans(): Promise<SavedPlanRecord[]>;
  savePlan(record: SavedPlanRecord): Promise<void>;
  deletePlan(id: string): Promise<void>;
  loadActiveDraft(): PlanInput | null;
  saveActiveDraft(input: PlanInput): void;
}

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE_PLANS)) {
        db.createObjectStore(STORE_PLANS, { keyPath: 'id' });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

class IndexedDBPlanRepository implements ClientPlanRepository {
  async listPlans(): Promise<SavedPlanRecord[]> {
    try {
      const db = await openDatabase();
      return await new Promise((resolve, reject) => {
        const tx = db.transaction(STORE_PLANS, 'readonly');
        const req = tx.objectStore(STORE_PLANS).getAll();
        req.onsuccess = () => {
          const items = (req.result as SavedPlanRecord[]) || [];
          items.sort((a, b) => b.updated_at.localeCompare(a.updated_at));
          resolve(items);
        };
        req.onerror = () => reject(req.error);
      });
    } catch {
      return [];
    }
  }

  async savePlan(record: SavedPlanRecord): Promise<void> {
    const db = await openDatabase();
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(STORE_PLANS, 'readwrite');
      tx.objectStore(STORE_PLANS).put(record);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
  }

  async deletePlan(id: string): Promise<void> {
    const db = await openDatabase();
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(STORE_PLANS, 'readwrite');
      tx.objectStore(STORE_PLANS).delete(id);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
  }

  loadActiveDraft(): PlanInput | null {
    try {
      // Check URL hash snapshot first (#plan=...)
      if (window.location.hash.startsWith('#plan=')) {
        const encoded = decodeURIComponent(window.location.hash.slice(6));
        const parsed = JSON.parse(atob(encoded)) as PlanInput;
        if (parsed && (parsed.country === 'IN' || parsed.country === 'US')) {
          return parsed;
        }
      }
      const raw = localStorage.getItem(ACTIVE_DRAFT_KEY);
      if (!raw) return null;
      return JSON.parse(raw) as PlanInput;
    } catch {
      return null;
    }
  }

  saveActiveDraft(input: PlanInput): void {
    try {
      localStorage.setItem(ACTIVE_DRAFT_KEY, JSON.stringify(input));
    } catch {
      // Ignore storage quota errors
    }
  }
}

export const planRepository: ClientPlanRepository = new IndexedDBPlanRepository();

export function encodeShareableURL(input: PlanInput): string {
  const json = JSON.stringify(input);
  const encoded = encodeURIComponent(btoa(json));
  return `${window.location.origin}${window.location.pathname}#plan=${encoded}`;
}
