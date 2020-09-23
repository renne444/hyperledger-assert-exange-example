package main

import (
	"github.com/gin-gonic/gin"
	"log"
)

func main(){
	router := gin.Default()

	router.GET("/hello", func(context *gin.Context){
		log.Println("Hello gin start")
		context.JSON(200,gin.H{
			"code":200,
			"success":true,
		})
	})

	router.Run("127.0.0.1:12587")
}