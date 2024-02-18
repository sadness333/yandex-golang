package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Knetic/govaluate"
	_ "github.com/lib/pq"
)

const (
	additionOperation       = "addition"
	subtractionOperation    = "subtraction"
	multiplicationOperation = "multiplication"
	divisionOperation       = "division"
	serverIdleOperation     = "server_idle"
)

func getDelay(operation string) (time.Duration, error) {
	var delay int
	err := db.QueryRow("SELECT delay_seconds FROM operation_delays WHERE operation = $1", operation).Scan(&delay)
	if err != nil {
		return 0, err
	}
	return time.Duration(delay) * time.Second, nil
}

func getServerIdleDelay() time.Duration {
	delay, _ := getDelay(serverIdleOperation)
	return delay
}

func calculateExpression(expression string) (float64, error) {
	expr, err := govaluate.NewEvaluableExpression(expression)
	if err != nil {
		return 0, err
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return 0, err
	}

	return result.(float64), nil
}

func getPendingTask() (Task, error) {
	// Получение задачи для выполнения из базы данных
	var task Task
	err := db.QueryRow("SELECT id, expression FROM tasks WHERE status = 'pending' ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED").
		Scan(&task.ID, &task.Expression)
	if err != nil {
		return Task{}, err
	}

	// Меняем статус задачи на "в процессе выполнения"
	_, err = db.Exec("UPDATE tasks SET status = 'in_progress' WHERE id = $1", task.ID)
	if err != nil {
		return Task{}, err
	}

	return task, nil
}

func agentBackgroundTask() {
	for {
		// Получение задачи для выполнения из базы данных
		task, err := getPendingTask()
		if err != nil {
			time.Sleep(getServerIdleDelay()) // Пауза перед повторной попыткой
			continue
		}

		// Выполнение вычислений и обновление задачи в базе данных
		delay, err := getDelay(task.Expression)
		if err != nil {
			log.Println("Error getting delay:", err)
			delay = getServerIdleDelay()
		}

		result, err := calculateAndSubmitResult(task)
		if err != nil {
			log.Println("Error calculating result:", err)
			time.Sleep(delay) // Пауза перед повторной попыткой
			continue
		}

		log.Printf("Task %d completed with result %.2f\n", task.ID, result)

		// Пауза перед следующей попыткой получить задачу
		time.Sleep(delay)
	}
}

func calculateAndSubmitResult(task Task) (float64, error) {
	// Реализация вычислений
	result, err := calculateExpression(task.Expression)
	if err != nil {
		fmt.Println("Ошибка здесь")
		return 0, err
	}

	// Обновляем задачу в базе данных с результатом и статусом "выполнено"
	_, err = db.Exec("UPDATE tasks SET result = $1, status = 'completed' WHERE id = $2", result, task.ID)
	if err != nil {
		// Обработка ошибок при обновлении задачи
		return 0, err
	}

	return result, nil
}
