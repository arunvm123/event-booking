package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arunvm123/eventbooking/booking-service/cache"
	"github.com/arunvm123/eventbooking/booking-service/model"
	"github.com/arunvm123/eventbooking/booking-service/repository"
	"github.com/arunvm123/eventbooking/booking-service/service"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Object pools for memory optimization
var (
	// Pool for booking request objects
	bookingRequestPool = sync.Pool{
		New: func() interface{} {
			return &model.BookingRequest{}
		},
	}

	// Pool for notification request objects
	notificationRequestPool = sync.Pool{
		New: func() interface{} {
			return &model.NotificationRequest{}
		},
	}

	// Pool for JSON encoding/decoding buffers
	jsonBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	// Pool for byte slices used in HTTP/JSON operations
	byteSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 4096) // 4KB initial capacity
		},
	}

	// Pool for status update objects
	statusUpdatePool = sync.Pool{
		New: func() interface{} {
			return &model.BookingStatusUpdate{}
		},
	}
)

// resetBookingRequest clears a booking request for reuse
func resetBookingRequest(req *model.BookingRequest) {
	req.BookingID = uuid.Nil
	req.UserID = uuid.Nil
	req.UserEmail = ""
	req.UserName = ""
	req.EventID = uuid.Nil
	req.EventName = ""
	req.Venue = ""
	req.EventDate = time.Time{}
	req.Seats = req.Seats[:0] // Keep capacity, reset length
	req.HoldID = uuid.Nil
	req.PaymentInfo = model.PaymentInfo{}
}

// resetNotificationRequest clears a notification request for reuse
func resetNotificationRequest(req *model.NotificationRequest) {
	req.Type = ""
	req.RecipientEmail = ""
	req.BookingData = model.NotificationBookingData{}
	req.Timestamp = time.Time{}
}

// resetStatusUpdate clears a status update for reuse
func resetStatusUpdate(update *model.BookingStatusUpdate) {
	update.BookingID = uuid.Nil
	update.Status = ""
	update.Message = ""
	update.UpdatedAt = time.Time{}
}

type BookingProcessor struct {
	repo         repository.BookingRepository
	cache        cache.CacheRepository
	eventService service.EventService
	kafkaWriter  *kafka.Writer
	consumer     *kafka.Reader

	// Worker pool for managing goroutines
	workerPool chan chan kafka.Message
	workers    []*BookingWorker

	// Metrics
	processedCount int64
	activeWorkers  int64
}

type BookingWorker struct {
	id         int
	processor  *BookingProcessor
	jobChannel chan kafka.Message
	workerPool chan chan kafka.Message
	quit       chan bool
}

func NewBookingProcessor(
	repo repository.BookingRepository,
	cache cache.CacheRepository,
	eventService service.EventService,
	kafkaWriter *kafka.Writer,
	consumer *kafka.Reader,
) *BookingProcessor {
	// Worker pool configuration
	maxWorkers := 20

	processor := &BookingProcessor{
		repo:         repo,
		cache:        cache,
		eventService: eventService,
		kafkaWriter:  kafkaWriter,
		consumer:     consumer,
		workerPool:   make(chan chan kafka.Message, maxWorkers),
		workers:      make([]*BookingWorker, maxWorkers),
	}

	// Initialize worker pool
	for i := 0; i < maxWorkers; i++ {
		worker := &BookingWorker{
			id:         i,
			processor:  processor,
			jobChannel: make(chan kafka.Message),
			workerPool: processor.workerPool,
			quit:       make(chan bool),
		}
		processor.workers[i] = worker
	}

	return processor
}

// Start begins processing booking requests from Kafka
func (p *BookingProcessor) Start(ctx context.Context) error {
	log.Printf("Starting booking processor with %d workers...", len(p.workers))

	// Start all workers
	for _, worker := range p.workers {
		worker.start()
	}

	// Start metrics reporting goroutine
	go p.reportMetrics(ctx)

	// Main message processing loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Booking processor shutting down...")
			p.shutdown()
			return ctx.Err()
		default:
			// Read message from Kafka
			msg, err := p.consumer.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Dispatch to worker pool (blocks if all workers busy)
			select {
			case jobChannel := <-p.workerPool:
				// Send job to available worker
				select {
				case jobChannel <- msg:
					// Successfully dispatched
				case <-ctx.Done():
					return ctx.Err()
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// BookingWorker methods
func (w *BookingWorker) start() {
	go func() {
		for {
			// Register this worker in the pool
			w.workerPool <- w.jobChannel

			select {
			case job := <-w.jobChannel:
				// Process the booking
				atomic.AddInt64(&w.processor.activeWorkers, 1)

				if err := w.processor.processBooking(job); err != nil {
					log.Printf("Worker %d error processing booking: %v", w.id, err)
				}

				atomic.AddInt64(&w.processor.processedCount, 1)
				atomic.AddInt64(&w.processor.activeWorkers, -1)

			case <-w.quit:
				log.Printf("Worker %d shutting down", w.id)
				return
			}
		}
	}()
}

func (w *BookingWorker) stop() {
	w.quit <- true
}

// shutdown gracefully stops all workers
func (p *BookingProcessor) shutdown() {
	log.Println("Shutting down booking processor workers...")

	for _, worker := range p.workers {
		worker.stop()
	}

	// Wait for active workers to finish (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			log.Println("Shutdown timeout reached, forcing exit")
			return
		case <-ticker.C:
			if atomic.LoadInt64(&p.activeWorkers) == 0 {
				log.Println("All workers finished gracefully")
				return
			}
		}
	}
}

// reportMetrics logs performance metrics
func (p *BookingProcessor) reportMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed := atomic.LoadInt64(&p.processedCount)
			active := atomic.LoadInt64(&p.activeWorkers)
			log.Printf("Booking Processor Metrics - Processed: %d, Active Workers: %d",
				processed, active)
		}
	}
}

// processBooking handles individual booking requests with object pooling
func (p *BookingProcessor) processBooking(msg kafka.Message) error {
	// Get pooled booking request object
	bookingReq := bookingRequestPool.Get().(*model.BookingRequest)
	defer func() {
		resetBookingRequest(bookingReq)
		bookingRequestPool.Put(bookingReq)
	}()

	// Unmarshal into pooled object
	if err := json.Unmarshal(msg.Value, bookingReq); err != nil {
		return fmt.Errorf("failed to unmarshal booking request: %w", err)
	}

	log.Printf("Processing booking: %s for user: %s", bookingReq.BookingID, bookingReq.UserID)

	// Update status to processing
	p.updateBookingStatus(bookingReq.BookingID, "processing", "payment", "Processing payment...", nil, nil)

	// Step 1: Simulate payment processing
	if err := p.processPayment(*bookingReq); err != nil {
		// Payment failed - release hold and mark booking as failed
		p.eventService.ReleaseHold(bookingReq.HoldID)
		failTime := time.Now()
		errMsg := fmt.Sprintf("Payment failed: %s", err.Error())
		p.updateBookingStatus(bookingReq.BookingID, "failed", "failed", errMsg, nil, &failTime)
		p.sendNotification(*bookingReq, "booking_failed", errMsg)
		return err
	}

	// Step 2: Confirm hold with Event Service (mark seats as booked)
	if err := p.eventService.ConfirmHold(bookingReq.HoldID); err != nil {
		// Hold confirmation failed - could be expired, seats taken, etc.
		failTime := time.Now()
		errMsg := fmt.Sprintf("Failed to confirm seats: %s", err.Error())
		p.updateBookingStatus(bookingReq.BookingID, "failed", "refund_pending", errMsg, nil, &failTime)
		p.sendNotification(*bookingReq, "booking_failed", errMsg)
		return err
	}

	// Step 3: Mark booking as confirmed
	confirmTime := time.Now()
	p.updateBookingStatus(bookingReq.BookingID, "confirmed", "completed", "Booking confirmed successfully", &confirmTime, nil)

	// Step 4: Send confirmation notification
	p.sendNotification(*bookingReq, "booking_confirmed", "Your booking has been confirmed!")

	log.Printf("Successfully processed booking: %s", bookingReq.BookingID)
	return nil
}

// processPayment simulates payment processing
func (p *BookingProcessor) processPayment(bookingReq model.BookingRequest) error {
	// Simulate payment processing time
	time.Sleep(2 * time.Second)

	// Simulate payment validation
	if bookingReq.PaymentInfo.Amount <= 0 {
		return fmt.Errorf("invalid payment amount: %f", bookingReq.PaymentInfo.Amount)
	}

	if bookingReq.PaymentInfo.PaymentMethod == "" {
		return fmt.Errorf("payment method is required")
	}

	// Simulate payment failure rate (5% failure for demo)
	// In real implementation, this would call actual payment gateway
	if time.Now().UnixNano()%20 == 0 {
		return fmt.Errorf("payment gateway declined transaction")
	}

	log.Printf("Payment processed successfully for booking: %s, amount: $%.2f",
		bookingReq.BookingID, bookingReq.PaymentInfo.Amount)
	return nil
}

// updateBookingStatus updates booking status in both database and cache
func (p *BookingProcessor) updateBookingStatus(bookingID uuid.UUID, status, paymentStatus, message string, confirmedAt, failedAt *time.Time) {
	// Update database
	updateReq := model.UpdateBookingStatusRequest{
		BookingID:     bookingID,
		Status:        status,
		PaymentStatus: paymentStatus,
		ConfirmedAt:   confirmedAt,
		FailedAt:      failedAt,
	}

	if status == "failed" {
		updateReq.ErrorMessage = &message
	}

	if err := p.repo.UpdateBookingStatus(updateReq); err != nil {
		log.Printf("Failed to update booking status in database: %v", err)
	}

	// Update cache for SSE
	statusUpdate := &model.BookingStatusUpdate{
		BookingID: bookingID,
		Status:    status,
		Message:   message,
		UpdatedAt: time.Now(),
	}

	if err := p.cache.SetBookingStatus(bookingID, statusUpdate, 24*time.Hour); err != nil {
		log.Printf("Failed to update booking status in cache: %v", err)
	}
}

// sendNotification sends notification to Kafka notification topic with object pooling
func (p *BookingProcessor) sendNotification(bookingReq model.BookingRequest, notificationType, message string) {
	// Get pooled notification request object
	notification := notificationRequestPool.Get().(*model.NotificationRequest)
	defer func() {
		resetNotificationRequest(notification)
		notificationRequestPool.Put(notification)
	}()

	// Get pooled JSON buffer
	jsonBuffer := jsonBufferPool.Get().(*bytes.Buffer)
	defer func() {
		jsonBuffer.Reset()
		jsonBufferPool.Put(jsonBuffer)
	}()

	// Populate notification
	notification.Type = notificationType
	notification.RecipientEmail = bookingReq.UserEmail
	notification.BookingData = model.NotificationBookingData{
		BookingID:   bookingReq.BookingID,
		EventName:   bookingReq.EventName,
		Venue:       bookingReq.Venue,
		EventDate:   bookingReq.EventDate,
		Seats:       bookingReq.Seats,
		TotalAmount: bookingReq.PaymentInfo.Amount,
		UserName:    bookingReq.UserName,
	}
	notification.Timestamp = time.Now()

	// Encode using pooled buffer
	encoder := json.NewEncoder(jsonBuffer)
	if err := encoder.Encode(notification); err != nil {
		log.Printf("Failed to encode notification: %v", err)
		return
	}

	p.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(bookingReq.BookingID.String()),
			Value: jsonBuffer.Bytes(),
		})
}
