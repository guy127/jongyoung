// รูปแบบข้อมูลจาก API (ตรงกับ DTO ฝั่ง Go) — เวลาทุกตัวเป็น timestamp เต็มพร้อม offset

export type Slot = { start_at: string; end_at: string; available: number; closed?: boolean };

export type Rating = { average: number | null; count: number };

export type RestaurantImage = { id: string; url: string; sort_order: number };

export type Restaurant = {
  id: string;
  owner_id: string;
  name: string;
  description: string;
  cuisine: string;
  address: string;
  seats: number;
  open_time: string;
  close_time: string;
  overnight: boolean;
  open_24h: boolean;
  closed_weekdays: number[]; // วันปิดประจำสัปดาห์ของวันทำการ (0 = อาทิตย์ … 6 = เสาร์)
  cancel_before_minutes: number;
  rating: Rating;
  images: RestaurantImage[];
  created_at: string;
};

export type ListItem = Restaurant & { slots?: Slot[] };

export type Page<T> = { items: T[]; page: number; limit: number; total: number };

export type Availability = { business_date: string; closed: boolean; opens_at: string; closes_at: string; seats: number; slots: Slot[] };

export type NextAvailable = { business_date: string; slots: Slot[] } | null;

export type Booking = {
  id: string;
  code: string;
  restaurant: { id: string; name: string; address: string; overnight: boolean };
  customer_name?: string;
  party_size: number;
  start_at: string;
  end_at: string;
  business_date: string;
  status: "active" | "cancelled";
  cancelled_at?: string;
  cancel_until: string;
  can_change: boolean;
  created_at: string;
};

export type Board = {
  business_date: string;
  closed: boolean; // วันปิดประจำสัปดาห์
  opens_at: string;
  closes_at: string;
  seats: number;
  slots: { start_at: string; end_at: string; booked: number }[];
  bookings: Booking[];
};

export type Review = { id: string; author_name?: string; score: number; body: string; created_at: string; updated_at: string };

export type Me = { id: string; email: string; display_name: string; restaurants: { id: string; name: string }[] };

export type ApiError = { code: string; message: string; details?: Record<string, unknown> };
