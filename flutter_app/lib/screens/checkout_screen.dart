import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:razorpay_flutter/razorpay_flutter.dart';
import '../providers/cart_provider.dart';
import '../services/payment_service.dart';
import '../services/api_service.dart';

/// Checkout screen handling order creation and payment flow.
/// Implements proper Razorpay integration with backend verification.
class CheckoutScreen extends ConsumerStatefulWidget {
  const CheckoutScreen({super.key});

  @override
  ConsumerState<CheckoutScreen> createState() => _CheckoutScreenState();
}

class _CheckoutScreenState extends ConsumerState<CheckoutScreen> {
  late PaymentService _paymentService;
  bool _isProcessing = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    final apiService = ref.read(apiServiceProvider);
    _paymentService = PaymentService(apiService: apiService);
  }

  @override
  void dispose() {
    _paymentService.dispose();
    super.dispose();
  }

  /// Initiate checkout process
  Future<void> _startCheckout() async {
    final cartState = ref.read(cartProvider);
    if (cartState.isEmpty) {
      _showSnackBar('Cart is empty', isError: true);
      return;
    }

    setState(() {
      _isProcessing = true;
      _error = null;
    });

    try {
      // Step 1: Create order on backend
      final apiService = ref.read(apiServiceProvider);
      final orderResponse = await apiService.createOrder(cartState.items);

      // Step 2: Start Razorpay payment flow
      await _paymentService.startPayment(
        orderDetails: orderResponse,
        userPhone: '9999999999', // Get from user profile
        userEmail: 'user@example.com', // Get from user profile
        onSuccess: _handlePaymentSuccess,
        onFailure: _handlePaymentFailure,
      );
    } on ApiException catch (e) {
      setState(() {
        _error = e.message;
        _isProcessing = false;
      });
      _showSnackBar(e.message, isError: true);
    } catch (e) {
      setState(() {
        _error = 'Failed to create order. Please try again.';
        _isProcessing = false;
      });
      _showSnackBar(_error!, isError: true);
    }
  }

  /// Handle Razorpay success callback.
  /// CRITICAL: Do NOT assume payment is successful here.
  /// Must verify with backend using signature.
  void _handlePaymentSuccess(PaymentSuccessResponse response) async {
    debugPrint('Payment success callback - verifying with backend...');

    try {
      // CRITICAL: Verify payment with backend
      final result = await _paymentService.verifyPayment(
        orderId: _paymentService.currentOrderId!,
        razorpayOrderId: response.orderId!,
        razorpayPaymentId: response.paymentId!,
        razorpaySignature: response.signature!,
      );

      if (result.success) {
        // Payment confirmed by backend - clear cart and show success
        ref.read(cartProvider.notifier).clearCart();
        
        setState(() {
          _isProcessing = false;
        });

        _showSnackBar('Payment successful! Order confirmed.', isError: false);
        
        // Navigate to order confirmation
        if (mounted) {
          Navigator.of(context).pushReplacementNamed(
            '/order-confirmation',
            arguments: result.orderId,
          );
        }
      } else {
        // Backend rejected payment
        setState(() {
          _error = result.message;
          _isProcessing = false;
        });
        _showSnackBar('Payment verification failed: ${result.message}', isError: true);
      }
    } catch (e) {
      setState(() {
        _error = 'Payment verification failed. Please contact support.';
        _isProcessing = false;
      });
      _showSnackBar(_error!, isError: true);
    }
  }

  /// Handle payment failure
  void _handlePaymentFailure(PaymentFailureResponse response) {
    debugPrint('Payment failed: ${response.code} - ${response.message}');

    setState(() {
      _error = response.message ?? 'Payment failed. Please try again.';
      _isProcessing = false;
    });

    _showSnackBar(_error!, isError: true);
  }

  /// Show snackbar message
  void _showSnackBar(String message, {required bool isError}) {
    if (!mounted) return;
    
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: isError ? Colors.red : Colors.green,
        behavior: SnackBarBehavior.floating,
        duration: Duration(seconds: isError ? 4 : 2),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final cartState = ref.watch(cartProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Checkout'),
        backgroundColor: Colors.orange,
        foregroundColor: Colors.white,
      ),
      body: cartState.isEmpty
          ? const Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.shopping_cart_outlined, size: 64, color: Colors.grey),
                  SizedBox(height: 16),
                  Text('Your cart is empty', style: TextStyle(fontSize: 18)),
                ],
              ),
            )
          : Column(
              children: [
                // Cart items list
                Expanded(
                  child: ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: cartState.items.length,
                    itemBuilder: (context, index) {
                      final item = cartState.items[index];
                      return Card(
                        margin: const EdgeInsets.only(bottom: 12),
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Row(
                            children: [
                              // Item image placeholder
                              Container(
                                width: 60,
                                height: 60,
                                decoration: BoxDecoration(
                                  color: Colors.orange.shade100,
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: const Icon(Icons.fastfood, color: Colors.orange),
                              ),
                              const SizedBox(width: 12),
                              // Item details
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      item.menuItem.name,
                                      style: const TextStyle(
                                        fontWeight: FontWeight.bold,
                                        fontSize: 16,
                                      ),
                                    ),
                                    Text(
                                      item.menuItem.formattedPrice,
                                      style: TextStyle(color: Colors.grey.shade600),
                                    ),
                                  ],
                                ),
                              ),
                              // Quantity controls
                              Row(
                                children: [
                                  IconButton(
                                    icon: const Icon(Icons.remove_circle_outline),
                                    onPressed: _isProcessing
                                        ? null
                                        : () => ref
                                            .read(cartProvider.notifier)
                                            .removeItem(item.menuItem.id),
                                    color: Colors.orange,
                                  ),
                                  Text(
                                    '${item.quantity}',
                                    style: const TextStyle(
                                      fontSize: 16,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                  IconButton(
                                    icon: const Icon(Icons.add_circle_outline),
                                    onPressed: _isProcessing
                                        ? null
                                        : () => ref
                                            .read(cartProvider.notifier)
                                            .addItem(item.menuItem),
                                    color: Colors.orange,
                                  ),
                                ],
                              ),
                              // Subtotal
                              SizedBox(
                                width: 70,
                                child: Text(
                                  item.formattedSubtotal,
                                  textAlign: TextAlign.right,
                                  style: const TextStyle(fontWeight: FontWeight.bold),
                                ),
                              ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
                ),
                // Error message
                if (_error != null)
                  Container(
                    padding: const EdgeInsets.all(12),
                    margin: const EdgeInsets.symmetric(horizontal: 16),
                    decoration: BoxDecoration(
                      color: Colors.red.shade50,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Row(
                      children: [
                        const Icon(Icons.error_outline, color: Colors.red),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(_error!, style: const TextStyle(color: Colors.red)),
                        ),
                      ],
                    ),
                  ),
                // Order summary and pay button
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    boxShadow: [
                      BoxShadow(
                        color: Colors.grey.shade300,
                        blurRadius: 10,
                        offset: const Offset(0, -5),
                      ),
                    ],
                  ),
                  child: SafeArea(
                    child: Column(
                      children: [
                        // Order summary
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(
                              'Total (${cartState.itemCount} items)',
                              style: const TextStyle(fontSize: 16),
                            ),
                            Text(
                              cartState.formattedTotal,
                              style: const TextStyle(
                                fontSize: 20,
                                fontWeight: FontWeight.bold,
                                color: Colors.orange,
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 16),
                        // Pay button
                        SizedBox(
                          width: double.infinity,
                          height: 50,
                          child: ElevatedButton(
                            onPressed: _isProcessing ? null : _startCheckout,
                            style: ElevatedButton.styleFrom(
                              backgroundColor: Colors.orange,
                              foregroundColor: Colors.white,
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                            ),
                            child: _isProcessing
                                ? const SizedBox(
                                    width: 24,
                                    height: 24,
                                    child: CircularProgressIndicator(
                                      color: Colors.white,
                                      strokeWidth: 2,
                                    ),
                                  )
                                : const Text(
                                    'Pay Now',
                                    style: TextStyle(
                                      fontSize: 18,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
    );
  }
}
