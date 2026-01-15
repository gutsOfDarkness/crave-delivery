import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:video_player/video_player.dart';
import 'dart:html' as html;
import '../models/menu_item.dart';
import '../providers/cart_provider.dart';

/// Provider for menu items with loading state
final menuProvider = FutureProvider<List<MenuItem>>((ref) async {
  final apiService = ref.read(apiServiceProvider);
  return await apiService.getMenu();
});

/// Provider for selected category
final selectedCategoryProvider = StateProvider<String?>((ref) => null);

/// Provider for expanded categories (showing all items)
final expandedCategoriesProvider = StateProvider<Set<String>>((ref) => {});

/// Provider for video player controller
final videoControllerProvider = StateNotifierProvider<VideoControllerNotifier, VideoPlayerController?>((ref) {
  return VideoControllerNotifier();
});

class VideoControllerNotifier extends StateNotifier<VideoPlayerController?> {
  VideoControllerNotifier() : super(null) {
    _initializeVideo();
  }

  Future<void> _initializeVideo() async {
    try {
      // Get the current host from the browser window
      final host = html.window.location.hostname ?? 'localhost';
      final port = 8888;
      final videoUrl = 'http://$host:$port/restaurant.mp4';
      
      final controller = VideoPlayerController.networkUrl(
        Uri.parse(videoUrl),
      );
      await controller.initialize();
      controller.setLooping(true);
      controller.play();
      state = controller;
    } catch (e) {
      print('Error initializing video: $e');
    }
  }

  @override
  void dispose() {
    state?.dispose();
    super.dispose();
  }
}

/// Menu screen displaying available food items.
/// Implements optimistic cart updates with immediate UI feedback.
class MenuScreen extends ConsumerWidget {
  const MenuScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final menuAsync = ref.watch(menuProvider);
    final cartState = ref.watch(cartProvider);
    final selectedCategory = ref.watch(selectedCategoryProvider);
    final expandedCategories = ref.watch(expandedCategoriesProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Food Menu'),
        backgroundColor: Colors.purple.shade300,
        foregroundColor: Colors.white,
        actions: [
          // Cart button with badge
          Stack(
            alignment: Alignment.center,
            children: [
              IconButton(
                icon: const Icon(Icons.shopping_cart),
                onPressed: () => Navigator.pushNamed(context, '/checkout'),
              ),
              if (cartState.itemCount > 0)
                Positioned(
                  right: 8,
                  top: 8,
                  child: Container(
                    padding: const EdgeInsets.all(4),
                    decoration: const BoxDecoration(
                      color: Colors.red,
                      shape: BoxShape.circle,
                    ),
                    constraints: const BoxConstraints(minWidth: 18, minHeight: 18),
                    child: Text(
                      '${cartState.itemCount}',
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
      body: menuAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, stack) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.red),
              const SizedBox(height: 16),
              Text('Failed to load menu: $error'),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: () => ref.invalidate(menuProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (menuItems) {
          // Get unique categories
          final categories = <String>{};
          for (final item in menuItems) {
            categories.add(item.category);
          }
          final sortedCategories = categories.toList()..sort();

          return _buildCategoriesWithItems(
            context,
            ref,
            menuItems,
            sortedCategories,
            cartState,
            expandedCategories,
          );
        },
      ),
      bottomNavigationBar: cartState.isEmpty
          ? null
          : Container(
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
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            '${cartState.itemCount} items',
                            style: TextStyle(color: Colors.grey.shade600),
                          ),
                          Text(
                            cartState.formattedTotal,
                            style: const TextStyle(
                              fontSize: 18,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ],
                      ),
                    ),
                    ElevatedButton(
                      onPressed: () => Navigator.pushNamed(context, '/checkout'),
                      style: ElevatedButton.styleFrom(
                        backgroundColor: Colors.purple.shade300,
                        foregroundColor: Colors.white,
                        padding: const EdgeInsets.symmetric(
                          horizontal: 24,
                          vertical: 12,
                        ),
                      ),
                      child: const Text('View Cart'),
                    ),
                  ],
                ),
              ),
            ),
    );
  }

  Widget _buildCategoriesWithItems(
    BuildContext context,
    WidgetRef ref,
    List<MenuItem> allItems,
    List<String> categories,
    CartState cartState,
    Set<String> expandedCategories,
  ) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: categories.length + 1, // +1 for video section
      itemBuilder: (context, index) {
        // Video section at index 0
        if (index == 0) {
          return _buildVideoSection(context, ref);
        }

        // Adjust category index
        final categoryIndex = index - 1;
        final category = categories[categoryIndex];
        final categoryItems = allItems.where((item) => item.category == category).toList();
        final isExpanded = expandedCategories.contains(category);
        final displayItems = isExpanded ? categoryItems : categoryItems.take(3).toList();

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Category Card Header
            Container(
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(12),
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [
                    Colors.purple.shade400,
                    Colors.purple.shade600,
                  ],
                ),
                boxShadow: [
                  BoxShadow(
                    color: Colors.purple.shade300.withOpacity(0.3),
                    blurRadius: 12,
                    offset: const Offset(0, 6),
                  ),
                  BoxShadow(
                    color: Colors.purple.shade300.withOpacity(0.1),
                    blurRadius: 24,
                    offset: const Offset(0, 12),
                  ),
                ],
                border: Border.all(
                  color: Colors.purple.shade700,
                  width: 2,
                ),
              ),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                child: Row(
                  children: [
                    Icon(
                      _getCategoryIcon(category),
                      size: 32,
                      color: Colors.white,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        category,
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: Colors.white,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            // Items
            ...displayItems.map((item) => _buildMenuItem(context, ref, item, cartState)),
            // Show More button
            if (categoryItems.length > 3)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: SizedBox(
                  width: double.infinity,
                  child: OutlinedButton(
                    onPressed: () {
                      final notifier = ref.read(expandedCategoriesProvider.notifier);
                      if (isExpanded) {
                        notifier.state = expandedCategories..remove(category);
                      } else {
                        notifier.state = {...expandedCategories, category};
                      }
                    },
                    style: OutlinedButton.styleFrom(
                      side: BorderSide(color: Colors.purple.shade300, width: 2),
                      padding: const EdgeInsets.symmetric(vertical: 12),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    child: Text(
                      isExpanded ? 'Show Less' : 'Show More (${categoryItems.length - 3} more)',
                      style: TextStyle(
                        color: Colors.purple.shade300,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ),
              ),
            const SizedBox(height: 24),
          ],
        );
      },
    );
  }

  IconData _getCategoryIcon(String category) {
    switch (category) {
      case 'Breakfast':
        return Icons.breakfast_dining;
      case 'Fast Food':
        return Icons.fastfood;
      case 'Drinks':
        return Icons.local_drink;
      default:
        return Icons.restaurant;
    }
  }

  Widget _buildMenuItem(
    BuildContext context,
    WidgetRef ref,
    MenuItem item,
    CartState cartState,
  ) {
    final quantity = cartState.getQuantity(item.id);

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Item image
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: item.imageUrl != null
                  ? Image.network(
                      item.imageUrl!,
                      width: 80,
                      height: 80,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => _buildPlaceholderImage(),
                    )
                  : _buildPlaceholderImage(),
            ),
            const SizedBox(width: 12),
            // Item details
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    item.name,
                    style: const TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 16,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    item.description,
                    style: TextStyle(
                      color: Colors.grey.shade600,
                      fontSize: 13,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    item.formattedPrice,
                    style: TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 16,
                      color: Colors.purple.shade300,
                    ),
                  ),
                ],
              ),
            ),
            // Add to cart button / quantity control
            Column(
              children: [
                if (quantity == 0)
                  ElevatedButton(
                    onPressed: () {
                      // OPTIMISTIC UPDATE: UI updates immediately
                      ref.read(cartProvider.notifier).addItem(item);
                      _showAddedSnackBar(context, item.name);
                    },
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.purple.shade300,
                      foregroundColor: Colors.white,
                      minimumSize: const Size(80, 36),
                    ),
                    child: const Text('ADD'),
                  )
                else
                  Container(
                    decoration: BoxDecoration(
                      border: Border.all(color: Colors.purple.shade300),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          icon: const Icon(Icons.remove, size: 18),
                          onPressed: () {
                            ref.read(cartProvider.notifier).removeItem(item.id);
                          },
                          color: Colors.purple.shade300,
                          constraints: const BoxConstraints(
                            minWidth: 32,
                            minHeight: 32,
                          ),
                        ),
                        Text(
                          '$quantity',
                          style: const TextStyle(
                            fontWeight: FontWeight.bold,
                            fontSize: 16,
                          ),
                        ),
                        IconButton(
                          icon: const Icon(Icons.add, size: 18),
                          onPressed: () {
                            ref.read(cartProvider.notifier).addItem(item);
                          },
                          color: Colors.purple.shade300,
                          constraints: const BoxConstraints(
                            minWidth: 32,
                            minHeight: 32,
                          ),
                        ),
                      ],
                    ),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPlaceholderImage() {
    return Container(
      width: 80,
      height: 80,
      decoration: BoxDecoration(
        color: Colors.purple.shade100,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Icon(Icons.fastfood, color: Colors.purple.shade300, size: 32),
    );
  }

  void _showAddedSnackBar(BuildContext context, String itemName) {
    ScaffoldMessenger.of(context).clearSnackBars();
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('$itemName added to cart'),
        duration: const Duration(seconds: 1),
        behavior: SnackBarBehavior.floating,
        backgroundColor: Colors.green,
      ),
    );
  }

  Widget _buildVideoSection(BuildContext context, WidgetRef ref) {
    final videoController = ref.watch(videoControllerProvider);
    final screenHeight = MediaQuery.of(context).size.height;
    final videoHeight = screenHeight * 0.30; // 30% of screen height

    if (videoController == null || !videoController.value.isInitialized) {
      return Container(
        height: videoHeight,
        decoration: BoxDecoration(
          color: Colors.black,
          borderRadius: BorderRadius.circular(12),
        ),
        child: const Center(
          child: CircularProgressIndicator(color: Colors.purple),
        ),
      );
    }

    return Column(
      children: [
        ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: Container(
            height: videoHeight,
            color: Colors.black,
            child: Stack(
              alignment: Alignment.center,
              children: [
                VideoPlayer(videoController),
                if (!videoController.value.isPlaying)
                  Container(
                    decoration: BoxDecoration(
                      color: Colors.black54,
                      shape: BoxShape.circle,
                    ),
                    child: IconButton(
                      onPressed: () {
                        ref.read(videoControllerProvider.notifier);
                        videoController.play();
                      },
                      icon: const Icon(
                        Icons.play_arrow,
                        color: Colors.white,
                        size: 48,
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 20),
      ],
    );
  }
}
