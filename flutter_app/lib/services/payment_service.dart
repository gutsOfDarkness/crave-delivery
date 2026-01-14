import 'package:flutter/material.dart';
import 'package:razorpay_flutter/razorpay_flutter.dart';
import '../models/order.dart';
import 'api_service.dart';

/// Payment service handling Razorpay integration.
/// CRITICAL: Never assume payment success from client callback alone.
/// Always verify with backend using signature verification.
class PaymentService {
  final ApiService _apiService;
  late Razorpay _razorpay;

  // Callbacks for payment events
  Function(PaymentSuccessResponse)? _onSuccess;
  Function(PaymentFailureResponse)? _onFailure;
  Function(ExternalWalletResponse)? _onExternalWallet;

  // Store order details for verification
  String? _currentOrderId;

  PaymentService({required ApiService apiService}) : _apiService = apiService {
    _razorpay = Razorpay();
    _setupEventListeners();
  }

  /// Setup Razorpay event listeners
  void _setupEventListeners() {
    _razorpay.on(Razorpay.EVENT_PAYMENT_SUCCESS, _handlePaymentSuccess);
    _razorpay.on(Razorpay.EVENT_PAYMENT_ERROR, _handlePaymentError);
    _razorpay.on(Razorpay.EVENT_EXTERNAL_WALLET, _handleExternalWallet);
  }

  /// Handle successful payment callback from Razorpay SDK.
  /// IMPORTANT: This does NOT mean payment is confirmed!
  /// We must verify with our backend using the signature.
  void _handlePaymentSuccess(PaymentSuccessResponse response) {
    debugPrint('Razorpay success callback received');
    debugPrint('Payment ID: ${response.paymentId}');
    debugPrint('Order ID: ${response.orderId}');
    debugPrint('Signature: ${response.signature}');

    if (_onSuccess != null) {
      _onSuccess!(response);
    }
  }

  /// Handle payment failure
  void _handlePaymentError(PaymentFailureResponse response) {
    debugPrint('Razorpay error: ${response.code} - ${response.message}');

    if (_onFailure != null) {
      _onFailure!(response);
    }
  }

  /// Handle external wallet selection
  void _handleExternalWallet(ExternalWalletResponse response) {
    debugPrint('External wallet: ${response.walletName}');

    if (_onExternalWallet != null) {
      _onExternalWallet!(response);
    }
  }

  /// Start payment flow
  /// Returns the created order response for tracking
  Future<CreateOrderResponse> startPayment({
    required CreateOrderResponse orderDetails,
    required String userPhone,
    required String userEmail,
    required Function(PaymentSuccessResponse) onSuccess,
    required Function(PaymentFailureResponse) onFailure,
    Function(ExternalWalletResponse)? onExternalWallet,
  }) async {
    _currentOrderId = orderDetails.orderId;
    _onSuccess = onSuccess;
    _onFailure = onFailure;
    _onExternalWallet = onExternalWallet;

    // Configure Razorpay checkout options
    final options = {
      'key': orderDetails.razorpayKeyId,
      'amount': orderDetails.amount, // Amount in paisa
      'currency': orderDetails.currency,
      'name': orderDetails.name,
      'description': orderDetails.description,
      'order_id': orderDetails.razorpayOrderId,
      'prefill': {
        'contact': userPhone,
        'email': userEmail,
      },
      'theme': {
        'color': '#FF6B00', // Orange theme for food app
      },
      'retry': {
        'enabled': true,
        'max_count': 3,
      },
    };

    try {
      _razorpay.open(options);
      return orderDetails;
    } catch (e) {
      debugPrint('Error opening Razorpay: $e');
      rethrow;
    }
  }

  /// Verify payment with backend.
  /// CRITICAL: This is the only way to confirm payment success.
  /// Client-side callback can be spoofed.
  Future<PaymentVerificationResult> verifyPayment({
    required String orderId,
    required String razorpayOrderId,
    required String razorpayPaymentId,
    required String razorpaySignature,
  }) async {
    return await _apiService.verifyPayment(
      orderId: orderId,
      razorpayOrderId: razorpayOrderId,
      razorpayPaymentId: razorpayPaymentId,
      razorpaySignature: razorpaySignature,
    );
  }

  /// Get current order ID being processed
  String? get currentOrderId => _currentOrderId;

  /// Clean up resources
  void dispose() {
    _razorpay.clear();
  }
}

/// Payment result after full verification flow
class PaymentResult {
  final bool success;
  final String? orderId;
  final String? message;
  final String? errorCode;

  const PaymentResult({
    required this.success,
    this.orderId,
    this.message,
    this.errorCode,
  });

  factory PaymentResult.success(String orderId, String message) {
    return PaymentResult(
      success: true,
      orderId: orderId,
      message: message,
    );
  }

  factory PaymentResult.failure(String message, {String? errorCode}) {
    return PaymentResult(
      success: false,
      message: message,
      errorCode: errorCode,
    );
  }
}
