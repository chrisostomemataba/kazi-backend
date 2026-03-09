```
// ============================================
// MODULE 1: AUTHENTICATION & USER MANAGEMENT
// ============================================

Table users {
  id uuid [pk]
  phone_number varchar(13) [unique, not null]
  full_name varchar(100)
  profile_photo_url text [note: 'MinIO path']
  is_active boolean [default: true]
  is_phone_verified boolean [default: true]
  created_at timestamp [default: `now()`]
  updated_at timestamp [default: `now()`]
}

Table user_roles {
  id uuid [pk]
  user_id uuid [ref: > users.id, not null]
  role_type varchar(10) [not null, note: 'customer or maid']
  is_active boolean [default: true]
  created_at timestamp [default: `now()`]
  
  indexes {
    (user_id, role_type) [unique]
  }
}

Table otp_codes {
  id uuid [pk]
  phone_number varchar(13) [not null]
  code varchar(6) [not null]
  purpose varchar(20) [not null, note: 'registration, login, payout_confirm']
  is_used boolean [default: false]
  attempts integer [default: 0]
  expires_at timestamp [not null]
  created_at timestamp [default: `now()`]
  
  indexes {
    (phone_number, expires_at) [name: 'idx_otp_lookup']
  }
}

// ============================================
// MODULE 2: MAID MANAGEMENT
// ============================================

Table maid_profiles {
  id uuid [pk]
  user_id uuid [ref: - users.id, not null]
  bio text
  gender varchar(10) [note: 'male, female, other']
  date_of_birth date
  home_address text
  home_location_lat decimal(10,8)
  home_location_lng decimal(11,8)
  district varchar(50) [note: 'Kinondoni, Ilala, Temeke']
  ward varchar(50) [note: 'Mbezi, Mikocheni, etc']
  hourly_rate integer [not null, note: 'for regular bookings in TZS']
  offers_contracts boolean [default: false]
  monthly_contract_rate integer [note: 'if offers_contracts=true']
  verification_status varchar(20) [default: 'pending']
  id_number varchar(50)
  id_type varchar(20)
  rejection_reason text
  is_available_now boolean [default: true]
  verified_at timestamp
  created_at timestamp [default: `now()`]
  updated_at timestamp [default: `now()`]
  
  indexes {
    (verification_status) [name: 'idx_maid_status']
    (district, ward) [name: 'idx_maid_location']
    (offers_contracts) [name: 'idx_maid_contracts']
  }
}

Table maid_services {
  id uuid [pk]
  maid_id uuid [ref: > users.id, not null]
  service_type varchar(50) [not null, note: 'cleaning, cooking, laundry, childcare, ironing']
  
  indexes {
    (maid_id, service_type) [unique]
  }
}

Table maid_verification_documents {
  id uuid [pk]
  maid_id uuid [ref: > users.id, not null]
  document_type varchar(20) [not null, note: 'selfie_video, id_photo']
  file_url text [not null, note: 'MinIO path']
  uploaded_at timestamp [default: `now()`]
  
  indexes {
    (maid_id, document_type) [name: 'idx_maid_docs']
  }
}

Table maid_availability {
  id uuid [pk]
  maid_id uuid [ref: > users.id, not null]
  day_of_week integer [not null, note: '0=Sunday, 6=Saturday']
  start_time time [not null]
  end_time time [not null]
  is_active boolean [default: true]
  
  indexes {
    (maid_id, day_of_week) [name: 'idx_maid_schedule']
  }
}

Table maid_blocked_dates {
  id uuid [pk]
  maid_id uuid [ref: > users.id, not null]
  blocked_date date [not null]
  reason text
  created_at timestamp [default: `now()`]
  
  indexes {
    (maid_id, blocked_date) [name: 'idx_maid_blocks']
  }
}

Table maid_statistics {
  id uuid [pk]
  maid_id uuid [ref: - users.id, not null]
  average_rating decimal(2,1) [default: 0.0]
  total_reviews integer [default: 0]
  total_jobs_completed integer [default: 0]
  total_contracts_completed integer [default: 0]
  total_earnings bigint [default: 0]
  last_calculated_at timestamp [default: `now()`]
}

// ============================================
// MODULE 3A: CONTRACTS (NEW)
// ============================================

Table contracts {
  id uuid [pk]
  reference_number varchar(20) [unique, not null, note: 'CT202603080001']
  customer_id uuid [ref: > users.id, not null]
  maid_id uuid [ref: > users.id, not null]
  contract_type varchar(20) [not null, note: 'live_in, full_time_daily']
  monthly_rate integer [not null, note: 'in TZS']
  platform_commission_rate decimal(4,2) [not null]
  platform_commission_amount integer [not null]
  maid_payout_amount integer [not null]
  start_date date [not null]
  end_date date [not null]
  duration_months integer [not null]
  contract_status varchar(20) [default: 'pending_maid', note: 'pending_maid, active, completed, terminated_by_customer, terminated_by_maid']
  payment_status varchar(20) [default: 'unpaid', note: 'unpaid, paid_current_month, overdue']
  termination_reason text
  terminated_at timestamp
  created_at timestamp [default: `now()`]
  updated_at timestamp [default: `now()`]
  
  indexes {
    (customer_id, contract_status) [name: 'idx_customer_contracts']
    (maid_id, contract_status) [name: 'idx_maid_contracts']
    (contract_status, start_date) [name: 'idx_active_contracts']
  }
}

Table contract_payments {
  id uuid [pk]
  contract_id uuid [ref: > contracts.id, not null]
  payment_month date [not null, note: '2026-03-01 (1st of month)']
  monthly_rate integer [not null]
  platform_commission integer [not null]
  maid_payout integer [not null]
  payment_status varchar(20) [default: 'pending', note: 'pending, paid, failed, refunded']
  due_date date [not null]
  paid_at timestamp
  azampay_transaction_id varchar(100)
  created_at timestamp [default: `now()`]
  
  indexes {
    (contract_id, payment_month) [unique, name: 'idx_contract_monthly_payment']
    (payment_status, due_date) [name: 'idx_overdue_payments']
  }
}

// ============================================
// MODULE 3: BOOKING MANAGEMENT
// ============================================

Table bookings {
  id uuid [pk]
  reference_number varchar(20) [unique, not null, note: 'BK202602080001']
  customer_id uuid [ref: > users.id, not null]
  maid_id uuid [ref: > users.id, not null]
  service_type varchar(50) [not null]
  booking_date date [not null]
  start_time time [not null]
  end_time time [not null]
  duration_hours decimal(3,1) [not null]
  special_instructions text
  booking_status varchar(30) [not null, default: 'pending_maid']
  payment_status varchar(20) [not null, default: 'unpaid']
  created_at timestamp [default: `now()`]
  updated_at timestamp [default: `now()`]
  
  indexes {
    (customer_id, booking_date) [name: 'idx_customer_bookings']
    (maid_id, booking_date) [name: 'idx_maid_bookings']
    (booking_status, booking_date) [name: 'idx_booking_status']
  }
}

Table booking_locations {
  id uuid [pk]
  booking_id uuid [ref: - bookings.id, not null]
  customer_address text [not null]
  customer_location_lat decimal(10,8)
  customer_location_lng decimal(11,8)
  district varchar(50)
  ward varchar(50)
  arrival_verified_lat decimal(10,8) [note: 'GPS when maid arrived']
  arrival_verified_lng decimal(11,8)
  arrival_verified_at timestamp
}

Table booking_pricing {
  id uuid [pk]
  booking_id uuid [ref: - bookings.id, not null]
  hourly_rate integer [not null, note: 'snapshot at booking time']
  subtotal_amount integer [not null]
  platform_commission_rate decimal(4,2) [not null]
  platform_commission_amount integer [not null]
  total_amount integer [not null]
  maid_payout_amount integer [not null]
}

Table booking_timeline {
  id uuid [pk]
  booking_id uuid [ref: > bookings.id, not null]
  event_type varchar(30) [not null, note: 'maid_accepted, payment_confirmed, service_started, service_completed, cancelled']
  event_timestamp timestamp [default: `now()`]
  triggered_by uuid [ref: > users.id]
  notes text
  
  indexes {
    (booking_id, event_timestamp) [name: 'idx_booking_timeline']
  }
}

// ============================================
// MODULE 4: PAYMENT MANAGEMENT
// ============================================

Table payments {
  id uuid [pk]
  booking_id uuid [ref: > bookings.id]
  user_id uuid [ref: > users.id, not null]
  transaction_type varchar(20) [not null, note: 'customer_checkout, maid_disbursement, refund']
  amount integer [not null, note: 'in TZS']
  provider varchar(20) [note: 'Mpesa, TigoPesa, AirtelMoney, Halopesa']
  account_number varchar(13)
  status varchar(20) [default: 'initiated']
  azampay_transaction_id varchar(100) [unique]
  azampay_reference varchar(100)
  failure_reason text
  raw_response jsonb [note: 'full AzamPay API response']
  initiated_at timestamp [default: `now()`]
  completed_at timestamp
  
  indexes {
    (booking_id) [name: 'idx_payment_booking']
    (azampay_transaction_id) [name: 'idx_payment_azampay']
    (status, initiated_at) [name: 'idx_payment_status']
  }
}

Table platform_escrow {
  id uuid [pk]
  booking_id uuid [ref: - bookings.id, not null]
  amount_held integer [not null]
  payment_id uuid [ref: > payments.id]
  status varchar(20) [default: 'holding']
  held_at timestamp [default: `now()`]
  released_at timestamp
}

Table maid_wallets {
  id uuid [pk]
  maid_id uuid [ref: - users.id, not null]
  available_balance integer [default: 0]
  pending_balance integer [default: 0]
  total_withdrawn bigint [default: 0]
  total_earned bigint [default: 0]
  last_withdrawal_at timestamp
  updated_at timestamp [default: `now()`]
}

Table wallet_transactions {
  id uuid [pk]
  maid_id uuid [ref: > users.id, not null]
  transaction_type varchar(30) [not null, note: 'credit, debit']
  amount integer [not null]
  balance_after integer [not null]
  related_booking_id uuid [ref: > bookings.id]
  related_payout_id uuid [ref: > payout_requests.id]
  description text
  created_at timestamp [default: `now()`]
  
  indexes {
    (maid_id, created_at) [name: 'idx_wallet_history']
  }
}

Table payout_requests {
  id uuid [pk]
  reference_number varchar(20) [unique, not null, note: 'PO202602080001']
  maid_id uuid [ref: > users.id, not null]
  amount_requested integer [not null]
  provider varchar(20) [not null]
  phone_number varchar(13) [not null]
  status varchar(20) [default: 'pending']
  azampay_transaction_id varchar(100)
  disbursement_payment_id uuid [ref: > payments.id]
  failure_reason text
  requested_at timestamp [default: `now()`]
  completed_at timestamp
  
  indexes {
    (maid_id, requested_at) [name: 'idx_payout_history']
  }
}

// ============================================
// MODULE 5: REVIEW & RATING
// ============================================

Table reviews {
  id uuid [pk]
  booking_id uuid [ref: - bookings.id, not null]
  reviewer_id uuid [ref: > users.id, not null]
  reviewee_id uuid [ref: > users.id, not null]
  rating integer [not null, note: '1 to 5']
  comment text
  is_visible boolean [default: true]
  created_at timestamp [default: `now()`]
  
  indexes {
    (reviewee_id, created_at) [name: 'idx_maid_reviews']
  }
}

Table review_tags {
  id uuid [pk]
  review_id uuid [ref: > reviews.id, not null]
  tag varchar(30) [not null, note: 'punctual, thorough, friendly, professional']
  
  indexes {
    (review_id) [name: 'idx_review_tags']
  }
}

// ============================================
// MODULE 6: COMMUNICATION
// ============================================

Table notifications {
  id uuid [pk]
  user_id uuid [ref: > users.id, not null]
  title varchar(100) [not null]
  message text [not null]
  notification_type varchar(30) [not null]
  related_booking_id uuid [ref: > bookings.id]
  is_read boolean [default: false]
  sent_via_sms boolean [default: false]
  created_at timestamp [default: `now()`]
  
  indexes {
    (user_id, is_read, created_at) [name: 'idx_user_notifications']
  }
}

Table in_app_messages {
  id uuid [pk]
  booking_id uuid [ref: > bookings.id, not null]
  sender_id uuid [ref: > users.id, not null]
  receiver_id uuid [ref: > users.id, not null]
  message_text text [not null]
  is_read boolean [default: false]
  created_at timestamp [default: `now()`]
  
  indexes {
    (booking_id, created_at) [name: 'idx_chat_messages']
  }
}

// ============================================
// MODULE 7: DISPUTE MANAGEMENT
// ============================================

Table disputes {
  id uuid [pk]
  booking_id uuid [ref: > bookings.id, not null]
  raised_by uuid [ref: > users.id, not null]
  dispute_type varchar(30) [not null]
  description text [not null]
  status varchar(30) [default: 'pending']
  refund_amount integer
  raised_at timestamp [default: `now()`]
  resolved_at timestamp
}

Table dispute_evidence {
  id uuid [pk]
  dispute_id uuid [ref: > disputes.id, not null]
  uploaded_by uuid [ref: > users.id, not null]
  file_url text [not null, note: 'MinIO path: /disputes/{booking_id}/evidence_{n}.jpg']
  uploaded_at timestamp [default: `now()`]
}

Table support_tickets {
  id uuid [pk]
  user_id uuid [ref: > users.id, not null]
  ticket_number varchar(20) [unique, not null]
  category varchar(30) [not null]
  subject varchar(200) [not null]
  description text [not null]
  status varchar(30) [default: 'open']
  admin_response text
  created_at timestamp [default: `now()`]
  resolved_at timestamp
}

// ============================================
// MODULE 8: ADMIN & AUDIT
// ============================================

Table admin_users {
  id uuid [pk]
  username varchar(50) [unique, not null]
  password_hash text [not null]
  full_name varchar(100)
  role varchar(20) [not null, note: 'super_admin, verifier, support_agent']
  is_active boolean [default: true]
  last_login_at timestamp
  created_at timestamp [default: `now()`]
}

Table audit_logs {
  id uuid [pk]
  actor_id uuid [note: 'admin_users.id or users.id']
  actor_type varchar(10) [not null, note: 'admin or user']
  action_type varchar(30) [not null]
  target_entity_type varchar(30) [note: 'booking, maid, payment, etc']
  target_entity_id uuid
  changes jsonb [note: 'old and new values']
  ip_address inet
  created_at timestamp [default: `now()`]
  
  indexes {
    (actor_id, created_at) [name: 'idx_audit_actor']
    (target_entity_type, target_entity_id) [name: 'idx_audit_target']
  }
}

Table platform_settings {
  id uuid [pk]
  setting_key varchar(50) [unique, not null, note: 'commission_rate, min_withdrawal_amount']
  setting_value text [not null]
  data_type varchar(10) [not null, note: 'integer, decimal, boolean, string']
  description text
  updated_at timestamp [default: `now()`]
}
```