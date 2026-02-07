/**
 * SSE (Server-Sent Events) Client for Stash.
 *
 * Replaces GraphQL WebSocket subscriptions for real-time updates.
 * Connects to /api/v1/events and dispatches events to registered listeners.
 */

import { getPlatformURL } from "./createClient";

const SSE_PATH = "/api/v1/events";

export type SSEEventType =
  | "connected"
  | "job.add"
  | "job.update"
  | "job.remove"
  | "log.entries"
  | "scan.complete";

export interface SSEJobStatusUpdate {
  type: "ADD" | "UPDATE" | "REMOVE";
  job: {
    id: string;
    status: string;
    subTasks?: string[];
    description: string;
    progress?: number;
    startTime?: string;
    endTime?: string;
    addTime: string;
    error?: string;
  };
}

export interface SSELogEntry {
  time: string;
  level: string;
  message: string;
}

type SSEListener<T = unknown> = (data: T) => void;

/**
 * SSEClient manages a persistent EventSource connection and
 * dispatches typed events to registered listeners.
 */
export class SSEClient {
  private eventSource: EventSource | null = null;
  private listeners: Map<string, Set<SSEListener>> = new Map();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private maxReconnectDelay = 30000;
  private _connected = false;

  get connected(): boolean {
    return this._connected;
  }

  /**
   * Connect to the SSE endpoint.
   */
  connect(): void {
    if (this.eventSource) {
      return; // already connected
    }

    const url = getPlatformURL(SSE_PATH).toString();
    // Don't use withCredentials in dev mode (cross-origin to :9999)
    // as it conflicts with Access-Control-Allow-Origin: *
    // In production, same-origin requests include credentials automatically
    this.eventSource = new EventSource(url);

    this.eventSource.onopen = () => {
      this._connected = true;
      this.reconnectDelay = 1000; // reset on successful connection
    };

    this.eventSource.onerror = () => {
      this._connected = false;
      this.cleanup();
      this.scheduleReconnect();
    };

    // Register known event types
    const eventTypes: SSEEventType[] = [
      "connected",
      "job.add",
      "job.update",
      "job.remove",
      "log.entries",
      "scan.complete",
    ];

    for (const type of eventTypes) {
      this.eventSource.addEventListener(type, (event: Event) => {
        const messageEvent = event as MessageEvent;
        let data: unknown;

        try {
          data = JSON.parse(messageEvent.data);
        } catch {
          data = messageEvent.data;
        }

        this.dispatch(type, data);
      });
    }
  }

  /**
   * Disconnect from the SSE endpoint.
   */
  disconnect(): void {
    this._connected = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.cleanup();
  }

  private cleanup(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, this.reconnectDelay);

    // Exponential backoff
    this.reconnectDelay = Math.min(
      this.reconnectDelay * 2,
      this.maxReconnectDelay
    );
  }

  /**
   * Subscribe to an SSE event type.
   * Returns an unsubscribe function.
   */
  on<T = unknown>(type: SSEEventType, listener: SSEListener<T>): () => void {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set());
    }

    const typedListener = listener as SSEListener;
    this.listeners.get(type)!.add(typedListener);

    return () => {
      this.listeners.get(type)?.delete(typedListener);
    };
  }

  /**
   * Subscribe to job status updates (add/update/remove).
   */
  onJobUpdate(listener: SSEListener<SSEJobStatusUpdate>): () => void {
    const unsubs = [
      this.on<SSEJobStatusUpdate>("job.add", listener),
      this.on<SSEJobStatusUpdate>("job.update", listener),
      this.on<SSEJobStatusUpdate>("job.remove", listener),
    ];
    return () => unsubs.forEach((u) => u());
  }

  /**
   * Subscribe to log entries.
   */
  onLogEntries(listener: SSEListener<SSELogEntry[]>): () => void {
    return this.on<SSELogEntry[]>("log.entries", listener);
  }

  /**
   * Subscribe to scan complete events.
   */
  onScanComplete(listener: SSEListener<void>): () => void {
    return this.on<void>("scan.complete", listener);
  }

  private dispatch(type: string, data: unknown): void {
    const typeListeners = this.listeners.get(type);
    if (typeListeners) {
      for (const listener of typeListeners) {
        try {
          listener(data);
        } catch (err) {
          console.error(`SSE listener error for event "${type}":`, err);
        }
      }
    }
  }
}

// Singleton instance
let sseInstance: SSEClient | null = null;

/**
 * Get the singleton SSE client instance.
 * Creates and connects on first call.
 */
export function getSSEClient(): SSEClient {
  if (!sseInstance) {
    sseInstance = new SSEClient();
    sseInstance.connect();
  }
  return sseInstance;
}
