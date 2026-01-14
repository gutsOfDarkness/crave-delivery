# Food Delivery Flutter App

Mobile app for the food delivery system with Razorpay payment integration.

## Tech Stack

- **Framework:** Flutter 3.x
- **State Management:** Riverpod
- **Payments:** razorpay_flutter
- **HTTP:** http package

## Features

### Optimistic UI Updates
Cart operations update the UI immediately before API calls complete. If an API call fails, the state rolls back and shows an error.

### Razorpay Integration
**CRITICAL:** Payment success from the Razorpay SDK callback is NOT trusted. The app always verifies payment with the backend using signature verification.

Flow:
1. Create order on backend → get Razorpay order ID
2. Open Razorpay checkout
3. On success callback → verify with backend `/orders/verify`
4. Only after backend confirms → show success to user

## Setup

### Prerequisites
- Flutter 3.x
- Android Studio / Xcode

### Install Dependencies

```bash
flutter pub get
```

### Configure API URL

Edit `lib/providers/cart_provider.dart`:

```dart
final apiServiceProvider = Provider<ApiService>((ref) {
  return ApiService(baseUrl: 'http://YOUR_BACKEND_URL:8080');
});
```

### Run

```bash
flutter run
```

## Project Structure

```
lib/
├── main.dart                    # App entry point
├── models/
│   ├── menu_item.dart          # Menu item model
│   ├── cart_item.dart          # Cart item model
│   └── order.dart              # Order models
├── providers/
│   └── cart_provider.dart      # Riverpod cart state
├── services/
│   ├── api_service.dart        # HTTP API client
│   └── payment_service.dart    # Razorpay wrapper
└── screens/
    ├── menu_screen.dart        # Menu listing
    ├── checkout_screen.dart    # Cart & payment
    └── order_confirmation_screen.dart
```

## Key Implementation Details

### Cart Provider (Optimistic Updates)

```dart
Future<void> addItem(MenuItem menuItem) async {
  // Save state for rollback
  _previousState = state;
  
  // Update UI immediately
  state = state.copyWith(items: updatedItems);
  
  // If API fails:
  // state = _previousState!.copyWith(error: 'Failed');
}
```

### Payment Verification

```dart
void _handlePaymentSuccess(PaymentSuccessResponse response) async {
  // NEVER assume success from client callback
  final result = await _paymentService.verifyPayment(
    orderId: orderId,
    razorpayOrderId: response.orderId!,
    razorpayPaymentId: response.paymentId!,
    razorpaySignature: response.signature!,
  );
  
  // Only trust backend verification result
  if (result.success) {
    // Payment confirmed
  }
}
```

## Android Setup

Add to `android/app/src/main/AndroidManifest.xml`:

```xml
<uses-permission android:name="android.permission.INTERNET"/>
```

## iOS Setup

No additional setup required for Razorpay.
